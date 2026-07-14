import {afterEach, describe, expect, it, vi} from 'vitest';
import {createApp} from './app.js';
import {ConfigurationError, readPort, readSmtpConfig} from './config.js';
import {buildMessageText} from './mail.js';
import {parseDemoRequest, RequestValidationError} from './validation.js';

const servers = [];
const pass = (_request, _response, next) => next();

async function request(app, path, init) {
    const server = app.listen(0, '127.0.0.1');
    servers.push(server);
    await new Promise((resolve) => server.once('listening', resolve));
    const {port} = server.address();
    return fetch(`http://127.0.0.1:${port}${path}`, init);
}

function demoBody(overrides = {}) {
    return JSON.stringify({
        name: '张三',
        company: '企智盒测试',
        phone: '13800138000',
        need: '测试预约',
        website: '',
        ...overrides,
    });
}

afterEach(async () => {
    await Promise.all(servers.splice(0).map((server) => new Promise((resolve, reject) => {
        server.close((error) => error ? reject(error) : resolve());
    })));
});

describe('服务配置', () => {
    it('读取 SMTP 和端口配置', () => {
        const env = {
            SMTP_HOST: 'smtp.example.com',
            SMTP_PORT: '465',
            SMTP_SECURE: 'true',
            SMTP_USER: 'sender@example.com',
            SMTP_PASS: 'secret',
            SMTP_FROM: 'sender@example.com',
            MAIL_TO: 'recipient@example.com',
            PORT: '3000',
        };

        expect(readSmtpConfig(env)).toMatchObject({host: 'smtp.example.com', port: 465, secure: true});
        expect(readPort(env)).toBe(3000);
    });

    it('拒绝无效配置', () => {
        expect(() => readSmtpConfig({})).toThrow(ConfigurationError);
        expect(() => readPort({PORT: 'invalid'})).toThrow(ConfigurationError);
    });
});

describe('预约字段和邮件', () => {
    it('清理字段并生成邮件正文', () => {
        const parsed = parseDemoRequest(demoBody({name: ' 张三 ', company: ' 企智盒测试 '}));
        expect(parsed).toMatchObject({name: '张三', company: '企智盒测试'});
        expect(buildMessageText(parsed, new Date('2026-07-14T08:00:00Z'))).toContain('联系电话：13800138000');
    });

    it('拒绝蜜罐字段和无效电话', () => {
        expect(() => parseDemoRequest(demoBody({website: 'https://bot.example'}))).toThrow(RequestValidationError);
        expect(() => parseDemoRequest(demoBody({phone: 'invalid'}))).toThrow(RequestValidationError);
    });
});

describe('预约接口', () => {
    it('发送有效预约', async () => {
        const mailSender = vi.fn().mockResolvedValue(undefined);
        const response = await request(createApp({mailSender, limiter: pass}), '/api/demo-requests', {
            method: 'POST',
            headers: {'Content-Type': 'application/json'},
            body: demoBody(),
        });

        expect(response.status).toBe(200);
        expect(await response.json()).toEqual({ok: true});
        expect(mailSender).toHaveBeenCalledOnce();
    });

    it('拒绝错误格式和非 POST 请求', async () => {
        const app = createApp({mailSender: vi.fn(), limiter: pass});
        const invalid = await request(app, '/api/demo-requests', {
            method: 'POST',
            headers: {'Content-Type': 'text/plain'},
            body: 'invalid',
        });
        const get = await request(app, '/api/demo-requests');

        expect(invalid.status).toBe(415);
        expect(get.status).toBe(405);
    });

    it('返回邮件发送失败', async () => {
        const response = await request(createApp({
            mailSender: vi.fn().mockRejectedValue(new Error('SMTP failed')),
            limiter: pass,
        }), '/api/demo-requests', {
            method: 'POST',
            headers: {'Content-Type': 'application/json'},
            body: demoBody(),
        });

        expect(response.status).toBe(502);
    });

    it('在限流后不发送邮件', async () => {
        const mailSender = vi.fn();
        const blocked = (_request, response) => {
            response.status(429).json({ok: false, error: '提交过于频繁，请稍后再试。'});
        };
        const response = await request(createApp({mailSender, limiter: blocked}), '/api/demo-requests', {
            method: 'POST',
            headers: {'Content-Type': 'application/json'},
            body: demoBody(),
        });

        expect(response.status).toBe(429);
        expect(mailSender).not.toHaveBeenCalled();
    });
});
