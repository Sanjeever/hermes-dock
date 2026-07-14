export interface RateLimiterBinding {
    limit(options: {key: string}): Promise<{success: boolean}>;
}

export interface WorkerEnv {
    SMTP_HOST?: string;
    SMTP_PORT?: string;
    SMTP_SECURE?: string;
    SMTP_USER?: string;
    SMTP_PASS?: string;
    SMTP_FROM?: string;
    MAIL_TO?: string;
    DEMO_RATE_LIMITER: RateLimiterBinding;
}

export interface SmtpConfig {
    host: string;
    port: number;
    secure: boolean;
    user: string;
    pass: string;
    from: string;
    to: string;
}

export class ConfigurationError extends Error {
    constructor(message: string) {
        super(message);
        this.name = 'ConfigurationError';
    }
}

const emailPattern = /^[^\s<>@]+@[^\s<>@]+\.[^\s<>@]+$/;

function required(env: WorkerEnv, key: keyof Pick<WorkerEnv, 'SMTP_HOST' | 'SMTP_PORT' | 'SMTP_SECURE' | 'SMTP_USER' | 'SMTP_PASS' | 'SMTP_FROM' | 'MAIL_TO'>): string {
    const value = env[key];
    if (typeof value !== 'string' || value.length === 0) {
        throw new ConfigurationError(`缺少 ${key} 配置。`);
    }
    return value;
}

function mailbox(value: string, key: 'SMTP_FROM' | 'MAIL_TO'): string {
    const normalized = value.trim();
    if (!emailPattern.test(normalized) || /[\r\n]/.test(normalized)) {
        throw new ConfigurationError(`${key} 必须是单个有效邮箱地址。`);
    }
    return normalized;
}

export function readSmtpConfig(env: WorkerEnv): SmtpConfig {
    const host = required(env, 'SMTP_HOST').trim();
    if (!host || host.length > 253 || /[\s/:]/.test(host)) {
        throw new ConfigurationError('SMTP_HOST 不是有效的 SMTP 主机名。');
    }

    const portValue = required(env, 'SMTP_PORT').trim();
    if (!/^\d+$/.test(portValue)) {
        throw new ConfigurationError('SMTP_PORT 必须是整数。');
    }
    const port = Number(portValue);
    if (port < 1 || port > 65535) {
        throw new ConfigurationError('SMTP_PORT 必须在 1 到 65535 之间。');
    }
    if (port === 25) {
        throw new ConfigurationError('Cloudflare Workers 不允许连接 SMTP 25 端口。');
    }

    const secureValue = required(env, 'SMTP_SECURE').trim().toLowerCase();
    if (secureValue !== 'true' && secureValue !== 'false') {
        throw new ConfigurationError('SMTP_SECURE 必须明确设置为 true 或 false。');
    }

    const user = required(env, 'SMTP_USER').trim();
    if (!user || /[\r\n]/.test(user)) {
        throw new ConfigurationError('SMTP_USER 不能为空或包含换行。');
    }

    return {
        host,
        port,
        secure: secureValue === 'true',
        user,
        pass: required(env, 'SMTP_PASS'),
        from: mailbox(required(env, 'SMTP_FROM'), 'SMTP_FROM'),
        to: mailbox(required(env, 'MAIL_TO'), 'MAIL_TO'),
    };
}
