import {createApp} from './app.js';
import {readPort, readSmtpConfig} from './config.js';
import {createMailSender} from './mail.js';

const port = readPort(process.env);
const smtpConfig = readSmtpConfig(process.env);
const app = createApp({mailSender: createMailSender(smtpConfig)});

const server = app.listen(port, '0.0.0.0', () => {
    console.log(`企智盒预约 API 已监听 0.0.0.0:${port}`);
});

function shutdown() {
    server.close((error) => {
        if (error) {
            console.error('预约服务关闭失败。');
            process.exitCode = 1;
        }
    });
}

process.once('SIGINT', shutdown);
process.once('SIGTERM', shutdown);
