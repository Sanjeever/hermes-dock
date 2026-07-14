import {readFile, rm, writeFile} from 'node:fs/promises';

const outlet = '<!--ssr-outlet-->';
const indexUrl = new URL('../dist/index.html', import.meta.url);
const serverDirectoryUrl = new URL('../dist-ssr/', import.meta.url);
const serverEntryUrl = new URL('entry-server.js', serverDirectoryUrl);
const template = await readFile(indexUrl, 'utf8');

if (!template.includes(outlet)) {
    throw new Error(`预渲染入口缺少 ${outlet}`);
}

const {render} = await import(serverEntryUrl.href);
const html = template.replace(outlet, () => render());

await writeFile(indexUrl, html);
await rm(serverDirectoryUrl, {recursive: true});
