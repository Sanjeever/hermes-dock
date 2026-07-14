import nodemailer from 'nodemailer';

export function buildMessageText(request, submittedAt = new Date()) {
    const submittedTime = new Intl.DateTimeFormat('zh-CN', {
        timeZone: 'Asia/Shanghai',
        dateStyle: 'medium',
        timeStyle: 'medium',
        hour12: false,
    }).format(submittedAt);

    return [
        '你收到一条新的企智盒预约演示申请。',
        '',
        `姓名：${request.name}`,
        `企业：${request.company}`,
        `联系电话：${request.phone}`,
        `需求：${request.need || '未填写'}`,
        `提交时间：${submittedTime}`,
    ].join('\n');
}

export function createMailSender(config) {
    const transporter = nodemailer.createTransport({
        host: config.host,
        port: config.port,
        secure: config.secure,
        requireTLS: !config.secure,
        auth: {
            user: config.user,
            pass: config.pass,
        },
        connectionTimeout: 15_000,
        greetingTimeout: 15_000,
        socketTimeout: 30_000,
    });

    return async (request) => {
        const result = await transporter.sendMail({
            from: {name: '企智盒官网', address: config.from},
            to: config.to,
            subject: '新的企智盒预约演示',
            text: buildMessageText(request),
        });

        if (result.accepted.length === 0 || result.rejected.length > 0) {
            throw new Error('SMTP 服务器未接受收件人。');
        }
    };
}
