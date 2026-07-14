import {connect} from 'cloudflare:sockets';
import type {SmtpConfig} from './config';
import type {DemoRequest} from './validation';

const responseTimeoutMs = 15_000;
const maxResponseBytes = 64 * 1024;
const clientHostname = 'qizhih-box.local';

type SmtpSocket = ReturnType<typeof connect>;

export interface SmtpReply {
    code: number;
    lines: string[];
}

export class SmtpError extends Error {
    constructor(readonly stage: string, readonly code?: number) {
        super(code ? `SMTP ${stage} 失败，服务器返回 ${code}。` : `SMTP ${stage} 失败。`);
        this.name = 'SmtpError';
    }
}

function timeout<T>(promise: Promise<T>, stage: string): Promise<T> {
    let timer: ReturnType<typeof setTimeout> | undefined;
    const expired = new Promise<never>((_, reject) => {
        timer = setTimeout(() => reject(new SmtpError(`${stage}超时`)), responseTimeoutMs);
    });
    return Promise.race([promise, expired]).finally(() => {
        if (timer !== undefined) clearTimeout(timer);
    });
}

function base64(value: string): string {
    const bytes = new TextEncoder().encode(value);
    let binary = '';
    for (const byte of bytes) binary += String.fromCharCode(byte);
    return btoa(binary);
}

function foldedBase64(value: string): string {
    return base64(value).match(/.{1,76}/g)?.join('\r\n') ?? '';
}

function encodedWord(value: string): string {
    return `=?UTF-8?B?${base64(value)}?=`;
}

function normalizeBody(value: string): string {
    return value.replace(/\r\n|\r/g, '\n');
}

export function buildMimeMessage(config: SmtpConfig, request: DemoRequest, submittedAt = new Date()): string {
    const submittedTime = new Intl.DateTimeFormat('zh-CN', {
        timeZone: 'Asia/Shanghai',
        dateStyle: 'medium',
        timeStyle: 'medium',
        hour12: false,
    }).format(submittedAt);
    const body = [
        '你收到一条新的企智盒预约演示申请。',
        '',
        `姓名：${request.name}`,
        `企业：${request.company}`,
        `联系电话：${request.phone}`,
        `需求：${request.need || '未填写'}`,
        `提交时间：${submittedTime}`,
    ].join('\n');
    const messageDomain = config.from.slice(config.from.lastIndexOf('@') + 1);

    return [
        `Date: ${submittedAt.toUTCString()}`,
        `Message-ID: <${crypto.randomUUID()}@${messageDomain}>`,
        `From: ${encodedWord('企智盒官网')} <${config.from}>`,
        `To: <${config.to}>`,
        `Subject: ${encodedWord('新的企智盒预约演示')}`,
        'MIME-Version: 1.0',
        'Content-Type: text/plain; charset=UTF-8',
        'Content-Transfer-Encoding: base64',
        '',
        foldedBase64(normalizeBody(body)),
        '',
    ].join('\r\n');
}

export function parseSmtpReply(lines: string[]): SmtpReply | null {
    if (!lines.length) return null;
    const first = /^(\d{3})([ -])/.exec(lines[0]);
    if (!first) throw new SmtpError('响应格式');

    const code = Number(first[1]);
    if (first[2] === ' ') return {code, lines};
    if (lines.length > 100) throw new SmtpError('响应过长');

    const last = lines[lines.length - 1];
    if (last.startsWith(`${code} `)) return {code, lines};
    return null;
}

class SmtpSession {
    private readonly decoder = new TextDecoder();
    private readonly encoder = new TextEncoder();
    private readonly reader: ReadableStreamDefaultReader<Uint8Array>;
    private readonly writer: WritableStreamDefaultWriter<Uint8Array>;
    private buffer = '';
    private released = false;

    constructor(private readonly socket: SmtpSocket) {
        this.reader = socket.readable.getReader();
        this.writer = socket.writable.getWriter();
    }

    async opened(): Promise<void> {
        await timeout(this.socket.opened.then(() => undefined), '连接');
    }

    private async readLine(): Promise<string> {
        while (true) {
            const newline = this.buffer.indexOf('\n');
            if (newline >= 0) {
                const line = this.buffer.slice(0, newline).replace(/\r$/, '');
                this.buffer = this.buffer.slice(newline + 1);
                return line;
            }

            const chunk = await timeout(this.reader.read(), '读取响应');
            if (chunk.done) throw new SmtpError('连接提前关闭');
            this.buffer += this.decoder.decode(chunk.value, {stream: true});
            if (this.buffer.length > maxResponseBytes) throw new SmtpError('响应过长');
        }
    }

    async reply(stage: string, expectedCodes: number[]): Promise<SmtpReply> {
        const lines: string[] = [];
        while (true) {
            lines.push(await this.readLine());
            const reply = parseSmtpReply(lines);
            if (!reply) continue;
            if (!expectedCodes.includes(reply.code)) throw new SmtpError(stage, reply.code);
            return reply;
        }
    }

    async command(command: string, stage: string, expectedCodes: number[]): Promise<SmtpReply> {
        await timeout(this.writer.write(this.encoder.encode(`${command}\r\n`)), `写入${stage}`);
        return this.reply(stage, expectedCodes);
    }

    async data(message: string): Promise<void> {
        const normalized = message
            .replace(/\r\n|\r|\n/g, '\n')
            .replace(/\n+$/, '')
            .split('\n')
            .map((line) => line.startsWith('.') ? `.${line}` : line)
            .join('\r\n');
        await timeout(this.writer.write(this.encoder.encode(`${normalized}\r\n.\r\n`)), '写入邮件正文');
        await this.reply('提交邮件', [250]);
    }

    release(): void {
        if (this.released) return;
        this.reader.releaseLock();
        this.writer.releaseLock();
        this.released = true;
    }
}

function capabilities(reply: SmtpReply): string[] {
    return reply.lines.map((line) => line.slice(4).trim().toUpperCase());
}

function authenticationMechanisms(serverCapabilities: string[]): Set<string> {
    const mechanisms = new Set<string>();
    for (const capability of serverCapabilities) {
        const match = /^AUTH(?:=|\s+)(.*)$/.exec(capability);
        if (!match) continue;
        for (const mechanism of match[1].split(/\s+/)) {
            if (mechanism) mechanisms.add(mechanism);
        }
    }
    return mechanisms;
}

async function authenticate(session: SmtpSession, config: SmtpConfig, serverCapabilities: string[]): Promise<void> {
    const mechanisms = authenticationMechanisms(serverCapabilities);
    if (mechanisms.has('LOGIN')) {
        await session.command('AUTH LOGIN', '开始认证', [334]);
        await session.command(base64(config.user), '认证用户名', [334]);
        await session.command(base64(config.pass), '认证密码', [235]);
        return;
    }
    if (mechanisms.has('PLAIN')) {
        const reply = await session.command(`AUTH PLAIN ${base64(`\0${config.user}\0${config.pass}`)}`, '认证', [235, 334]);
        if (reply.code === 334) await session.command(base64(`\0${config.user}\0${config.pass}`), '认证', [235]);
        return;
    }
    throw new SmtpError('服务器不支持 AUTH LOGIN 或 AUTH PLAIN');
}

export async function sendSmtpMail(config: SmtpConfig, request: DemoRequest): Promise<void> {
    let socket: SmtpSocket;
    try {
        socket = connect(
            {hostname: config.host, port: config.port},
            {secureTransport: config.secure ? 'on' : 'starttls', allowHalfOpen: false},
        );
    } catch {
        throw new SmtpError('连接');
    }
    let session = new SmtpSession(socket);

    try {
        await session.opened();
        await session.reply('服务器问候', [220]);
        let hello = await session.command(`EHLO ${clientHostname}`, 'EHLO', [250]);

        if (!config.secure) {
            const advertised = capabilities(hello);
            if (!advertised.some((capability) => capability === 'STARTTLS' || capability.startsWith('STARTTLS '))) {
                throw new SmtpError('服务器未提供 STARTTLS');
            }
            await session.command('STARTTLS', 'STARTTLS', [220]);
            session.release();
            socket = socket.startTls();
            session = new SmtpSession(socket);
            await session.opened();
            hello = await session.command(`EHLO ${clientHostname}`, 'TLS 后 EHLO', [250]);
        }

        await authenticate(session, config, capabilities(hello));
        await session.command(`MAIL FROM:<${config.from}>`, '设置发件人', [250]);
        await session.command(`RCPT TO:<${config.to}>`, '设置收件人', [250, 251]);
        await session.command('DATA', '开始邮件正文', [354]);
        await session.data(buildMimeMessage(config, request));

        try {
            await session.command('QUIT', '退出', [221]);
        } catch (error) {
            const detail = error instanceof SmtpError ? `${error.stage}${error.code ? ` (${error.code})` : ''}` : '未知错误';
            console.warn(`SMTP 邮件已被服务器接受，但 QUIT 失败：${detail}`);
        }
    } catch (error) {
        if (error instanceof SmtpError) throw error;
        throw new SmtpError('连接或传输');
    } finally {
        try {
            await socket.close();
        } catch {
            console.warn('SMTP 连接关闭失败。');
        }
        session.release();
    }
}
