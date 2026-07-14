import express from 'express';
import {rateLimit} from 'express-rate-limit';
import {maxRequestBytes, parseDemoRequest, RequestValidationError} from './validation.js';

const demoEndpoint = '/api/demo-requests';

function jsonError(message) {
    return {ok: false, error: message};
}

function createDemoLimiter() {
    return rateLimit({
        windowMs: 60_000,
        limit: 5,
        standardHeaders: 'draft-8',
        legacyHeaders: false,
        handler: (_request, response) => {
            response.status(429).json(jsonError('提交过于频繁，请稍后再试。'));
        },
    });
}

export function createApp({mailSender, limiter = createDemoLimiter()}) {
    const app = express();
    app.disable('x-powered-by');
    app.set('trust proxy', 1);

    app.use('/api', (_request, response, next) => {
        response.set({
            'Cache-Control': 'no-store',
            'X-Content-Type-Options': 'nosniff',
        });
        next();
    });

    app.get('/healthz', (_request, response) => {
        response.json({ok: true});
    });

    app.post(
        demoEndpoint,
        limiter,
        (request, response, next) => {
            const contentType = request.headers['content-type']?.split(';', 1)[0].trim().toLowerCase();
            if (contentType !== 'application/json') {
                response.status(415).json(jsonError('请求必须使用 application/json。'));
                return;
            }
            next();
        },
        express.text({type: 'application/json', limit: maxRequestBytes}),
        async (request, response) => {
            try {
                const demoRequest = parseDemoRequest(request.body);
                await mailSender(demoRequest);
                response.json({ok: true});
            } catch (error) {
                if (error instanceof RequestValidationError) {
                    response.status(error.status).json(jsonError(error.message));
                    return;
                }
                console.error('预约邮件发送失败。');
                response.status(502).json(jsonError('邮件服务器暂时无法发送预约通知。'));
            }
        },
    );

    app.all(demoEndpoint, (_request, response) => {
        response.set('Allow', 'POST').status(405).json(jsonError('仅支持 POST 请求。'));
    });

    app.use('/api', (_request, response) => {
        response.status(404).json(jsonError('接口不存在。'));
    });

    app.use((error, _request, response, next) => {
        if (error && typeof error === 'object' && error.type === 'entity.too.large') {
            response.status(413).json(jsonError('请求体过大。'));
            return;
        }
        if (response.headersSent) {
            next(error);
            return;
        }
        console.error('预约服务发生未知错误。');
        response.status(500).json(jsonError('预约服务暂时不可用。'));
    });

    return app;
}
