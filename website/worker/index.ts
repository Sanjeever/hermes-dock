import {ConfigurationError, readSmtpConfig, type SmtpConfig, type WorkerEnv} from './config';
import {sendSmtpMail, SmtpError} from './smtp';
import {maxRequestBytes, parseDemoRequest, RequestValidationError, type DemoRequest} from './validation';

type MailSender = (config: SmtpConfig, request: DemoRequest) => Promise<void>;

function json(body: Record<string, unknown>, status = 200, headers: HeadersInit = {}): Response {
    return Response.json(body, {
        status,
        headers: {
            'Cache-Control': 'no-store',
            'X-Content-Type-Options': 'nosniff',
            ...headers,
        },
    });
}

export async function handleDemoRequest(request: Request, env: WorkerEnv, mailSender: MailSender = sendSmtpMail): Promise<Response> {
    if (request.method !== 'POST') {
        return json({ok: false, error: '仅支持 POST 请求。'}, 405, {Allow: 'POST'});
    }

    const contentType = request.headers.get('Content-Type')?.split(';', 1)[0].trim().toLowerCase();
    if (contentType !== 'application/json') {
        return json({ok: false, error: '请求必须使用 application/json。'}, 415);
    }

    const declaredLength = Number(request.headers.get('Content-Length'));
    if (Number.isFinite(declaredLength) && declaredLength > maxRequestBytes) {
        return json({ok: false, error: '请求体过大。'}, 413);
    }

    try {
        const body = await request.text();
        if (new TextEncoder().encode(body).byteLength > maxRequestBytes) {
            throw new RequestValidationError('请求体过大。', 413);
        }
        const demoRequest = parseDemoRequest(body);

        const clientKey = request.headers.get('CF-Connecting-IP') ?? 'local';
        const rateLimit = await env.DEMO_RATE_LIMITER.limit({key: `demo:${clientKey}`});
        if (!rateLimit.success) {
            return json({ok: false, error: '提交过于频繁，请稍后再试。'}, 429);
        }

        const smtpConfig = readSmtpConfig(env);
        await mailSender(smtpConfig, demoRequest);
        return json({ok: true});
    } catch (error) {
        if (error instanceof RequestValidationError) {
            return json({ok: false, error: error.message}, error.status);
        }
        if (error instanceof ConfigurationError) {
            console.error(`预约邮件配置错误：${error.message}`);
            return json({ok: false, error: '预约服务尚未完成邮件配置。'}, 500);
        }
        if (error instanceof SmtpError) {
            console.error(`预约邮件发送失败：${error.stage}${error.code ? ` (${error.code})` : ''}`);
            return json({ok: false, error: '邮件服务器暂时无法发送预约通知。'}, 502);
        }
        console.error('预约接口发生未知错误。');
        return json({ok: false, error: '预约服务暂时不可用。'}, 500);
    }
}

export default {
    fetch(request, env): Promise<Response> {
        const url = new URL(request.url);
        if (url.pathname === '/api/demo-requests') return handleDemoRequest(request, env);
        return Promise.resolve(json({ok: false, error: '接口不存在。'}, 404));
    },
} satisfies ExportedHandler<WorkerEnv>;
