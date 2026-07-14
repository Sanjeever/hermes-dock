export class ConfigurationError extends Error {
    constructor(message) {
        super(message);
        this.name = 'ConfigurationError';
    }
}

const emailPattern = /^[^\s<>@]+@[^\s<>@]+\.[^\s<>@]+$/;

function required(env, key) {
    const value = env[key];
    if (typeof value !== 'string' || value.length === 0) {
        throw new ConfigurationError(`缺少 ${key} 配置。`);
    }
    return value;
}

function mailbox(value, key) {
    const normalized = value.trim();
    if (!emailPattern.test(normalized) || /[\r\n]/.test(normalized)) {
        throw new ConfigurationError(`${key} 必须是单个有效邮箱地址。`);
    }
    return normalized;
}

export function readSmtpConfig(env) {
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

export function readPort(env) {
    const value = env.PORT?.trim() || '3000';
    if (!/^\d+$/.test(value)) throw new ConfigurationError('PORT 必须是整数。');
    const port = Number(value);
    if (port < 1 || port > 65535) throw new ConfigurationError('PORT 必须在 1 到 65535 之间。');
    return port;
}
