import {describe, expect, it, vi} from 'vitest';
import {ConfigurationError, readSmtpConfig, type WorkerEnv} from './config';
import {handleDemoRequest} from './index';
import {buildMimeMessage, parseSmtpReply, SmtpError} from './smtp';
import {parseDemoRequest, RequestValidationError} from './validation';

function environment(): WorkerEnv {
    return {
        SMTP_HOST: 'smtp.example.com',
        SMTP_PORT: '465',
        SMTP_SECURE: 'true',
        SMTP_USER: 'sender@example.com',
        SMTP_PASS: 'secret',
        SMTP_FROM: 'sender@example.com',
        MAIL_TO: 'recipient@example.com',
        DEMO_RATE_LIMITER: {
            limit: vi.fn().mockResolvedValue({success: true}),
        },
    };
}

function demoRequest(body: Record<string, unknown> = {}): Request {
    return new Request('https://example.com/api/demo-requests', {
        method: 'POST',
        headers: {
            'Content-Type': 'application/json',
            'CF-Connecting-IP': '203.0.113.1',
        },
        body: JSON.stringify({
            name: '张三',
            company: '企智盒测试',
            phone: '13800138000',
            need: '测试预约',
            website: '',
            ...body,
        }),
    });
}

describe('SMTP 配置', () => {
    it('支持隐式 TLS 和 STARTTLS 配置', () => {
        expect(readSmtpConfig(environment())).toMatchObject({
            host: 'smtp.example.com',
            port: 465,
            secure: true,
        });

        const env = environment();
        env.SMTP_PORT = '587';
        env.SMTP_SECURE = 'false';
        expect(readSmtpConfig(env)).toMatchObject({port: 587, secure: false});
    });

    it('拒绝 Cloudflare 不允许的 SMTP 25 端口', () => {
        const env = environment();
        env.SMTP_PORT = '25';
        expect(() => readSmtpConfig(env)).toThrow(ConfigurationError);
    });

    it('拒绝无效发件地址', () => {
        const env = environment();
        env.SMTP_FROM = '企智盒 <sender@example.com>';
        expect(() => readSmtpConfig(env)).toThrow(ConfigurationError);
    });
});

describe('预约字段校验', () => {
    it('清理并返回有效字段', () => {
        expect(parseDemoRequest(JSON.stringify({
            name: ' 张三 ',
            company: ' 企智盒测试 ',
            phone: '13800138000',
            need: ' 测试预约 ',
            website: '',
        }))).toEqual({
            name: '张三',
            company: '企智盒测试',
            phone: '13800138000',
            need: '测试预约',
        });
    });

    it('拒绝蜜罐字段和无效电话', () => {
        expect(() => parseDemoRequest(JSON.stringify({
            name: '张三',
            company: '企智盒测试',
            phone: 'not-a-phone',
            website: 'https://bot.example',
        }))).toThrow(RequestValidationError);
    });
});

describe('SMTP 消息', () => {
    it('解析单行和多行 SMTP 响应', () => {
        expect(parseSmtpReply(['220 smtp.example.com ready'])).toMatchObject({code: 220});
        expect(parseSmtpReply(['250-smtp.example.com', '250 AUTH LOGIN PLAIN'])).toMatchObject({code: 250});
        expect(parseSmtpReply(['250-smtp.example.com'])).toBeNull();
        expect(() => parseSmtpReply(['invalid response'])).toThrow(SmtpError);
    });

    it('生成不暴露明文预约内容的 UTF-8 MIME 邮件', () => {
        const config = readSmtpConfig(environment());
        const message = buildMimeMessage(config, {
            name: '张三',
            company: '企智盒测试',
            phone: '13800138000',
            need: '测试预约',
        }, new Date('2026-07-14T08:00:00Z'));

        expect(message).toContain('Content-Transfer-Encoding: base64');
        expect(message).not.toContain('13800138000');
        const encodedBody = message.split('\r\n\r\n')[1].replace(/\r\n/g, '');
        const decodedBody = new TextDecoder().decode(Uint8Array.from(atob(encodedBody), (character) => character.charCodeAt(0)));
        expect(decodedBody).toContain('企业：企智盒测试');
    });
});

describe('预约接口', () => {
    it('校验、限流并调用邮件发送器', async () => {
        const send = vi.fn().mockResolvedValue(undefined);
        const response = await handleDemoRequest(demoRequest(), environment(), send);

        expect(response.status).toBe(200);
        expect(await response.json()).toEqual({ok: true});
        expect(send).toHaveBeenCalledOnce();
        expect(send.mock.calls[0][1]).toMatchObject({company: '企智盒测试'});
    });

    it('在限流后返回 429 且不发送邮件', async () => {
        const env = environment();
        env.DEMO_RATE_LIMITER.limit = vi.fn().mockResolvedValue({success: false});
        const send = vi.fn().mockResolvedValue(undefined);
        const response = await handleDemoRequest(demoRequest(), env, send);

        expect(response.status).toBe(429);
        expect(send).not.toHaveBeenCalled();
    });

    it('拒绝非 JSON 和无效字段', async () => {
        const env = environment();
        const plainResponse = await handleDemoRequest(new Request('https://example.com/api/demo-requests', {
            method: 'POST',
            headers: {'Content-Type': 'text/plain'},
            body: 'invalid',
        }), env);
        const invalidResponse = await handleDemoRequest(demoRequest({name: 'A'}), env);

        expect(plainResponse.status).toBe(415);
        expect(invalidResponse.status).toBe(400);
    });
});
