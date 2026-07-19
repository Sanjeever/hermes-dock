import {defineConfig} from 'vite';
import react from '@vitejs/plugin-react';
import {fileURLToPath} from 'node:url';

const projectRoot = fileURLToPath(new URL('.', import.meta.url));

export default defineConfig(({isSsrBuild}) => ({
    plugins: [react()],
    build: {
        rollupOptions: {
            input: isSsrBuild ? undefined : {
                main: `${projectRoot}index.html`,
                manual: `${projectRoot}manual/index.html`,
            },
        },
    },
    server: {
        proxy: {
            '/api': 'http://127.0.0.1:3000',
            '/healthz': 'http://127.0.0.1:3000',
        },
    },
}));
