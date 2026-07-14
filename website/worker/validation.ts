export const maxRequestBytes = 8 * 1024;

export interface DemoRequest {
    name: string;
    company: string;
    phone: string;
    need: string;
}

export class RequestValidationError extends Error {
    constructor(message: string, readonly status = 400) {
        super(message);
        this.name = 'RequestValidationError';
    }
}

function objectValue(value: unknown): Record<string, unknown> {
    if (typeof value !== 'object' || value === null || Array.isArray(value)) {
        throw new RequestValidationError('请求体必须是 JSON 对象。');
    }
    return value as Record<string, unknown>;
}

function requiredText(input: Record<string, unknown>, key: 'name' | 'company' | 'phone'): string {
    const value = input[key];
    if (typeof value !== 'string') {
        throw new RequestValidationError(`字段 ${key} 必须是字符串。`);
    }
    return value.trim();
}

export function parseDemoRequest(text: string): DemoRequest {
    let parsed: unknown;
    try {
        parsed = JSON.parse(text);
    } catch {
        throw new RequestValidationError('请求体不是有效 JSON。');
    }

    const input = objectValue(parsed);
    if (input.website !== undefined && typeof input.website !== 'string') {
        throw new RequestValidationError('无效的预约信息。');
    }
    if (typeof input.website === 'string' && input.website.trim()) {
        throw new RequestValidationError('无效的预约信息。');
    }

    const name = requiredText(input, 'name');
    const company = requiredText(input, 'company');
    const phone = requiredText(input, 'phone');
    const needValue = input.need ?? '';
    if (typeof needValue !== 'string') {
        throw new RequestValidationError('字段 need 必须是字符串。');
    }
    const need = needValue.trim();

    if (name.length < 2 || name.length > 50 || /[\u0000-\u001f\u007f]/.test(name)) {
        throw new RequestValidationError('姓名长度必须在 2 到 50 个字符之间。');
    }
    if (company.length < 2 || company.length > 100 || /[\u0000-\u001f\u007f]/.test(company)) {
        throw new RequestValidationError('企业名称长度必须在 2 到 100 个字符之间。');
    }
    if (!/^[0-9+()\-\s]{7,20}$/.test(phone)) {
        throw new RequestValidationError('联系电话格式不正确。');
    }
    if (need.length > 1000 || /\u0000/.test(need)) {
        throw new RequestValidationError('需求描述不能超过 1000 个字符。');
    }

    return {name, company, phone, need};
}
