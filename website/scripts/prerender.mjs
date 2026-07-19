import {readFile, rm, writeFile} from 'node:fs/promises';

const outlet = '<!--ssr-outlet-->';
const serverDirectoryUrl = new URL('../dist-ssr/', import.meta.url);
const serverEntryUrl = new URL('entry-server.js', serverDirectoryUrl);
const manualSourceUrl = new URL('../content/manual.md', import.meta.url);
const manualMeta = parseFrontMatter(await readFile(manualSourceUrl, 'utf8'));
const pages = [
    {pathname: '/', output: new URL('../dist/index.html', import.meta.url)},
    {pathname: '/manual/', output: new URL('../dist/manual/index.html', import.meta.url), meta: manualMeta},
];
const {render} = await import(serverEntryUrl.href);

for (const page of pages) {
    let template = await readFile(page.output, 'utf8');
    if (!template.includes(outlet)) {
        throw new Error(`${page.pathname} 预渲染入口缺少 ${outlet}`);
    }
    if (page.meta) template = injectManualMeta(template, page.meta);
    await writeFile(page.output, template.replace(outlet, () => render(page.pathname)));
}

await rm(serverDirectoryUrl, {recursive: true});

function parseFrontMatter(source) {
    const match = source.match(/^---\n([\s\S]*?)\n---\n/);
    if (!match) throw new Error('操作手册缺少 front matter');
    const meta = Object.fromEntries(match[1].split('\n').map((line) => {
        const separator = line.indexOf(':');
        return [line.slice(0, separator).trim(), line.slice(separator + 1).trim()];
    }));
    for (const key of ['title', 'description', 'updated']) {
        if (!meta[key]) throw new Error(`操作手册 front matter 缺少 ${key}`);
    }
    return meta;
}

function injectManualMeta(template, meta) {
    const replacements = {
        '__MANUAL_TITLE_HTML__': escapeHTML(meta.title),
        '__MANUAL_DESCRIPTION_HTML__': escapeHTML(meta.description),
        '__MANUAL_TITLE_JSON__': JSON.stringify(meta.title),
        '__MANUAL_DESCRIPTION_JSON__': JSON.stringify(meta.description),
        '__MANUAL_UPDATED_JSON__': JSON.stringify(meta.updated),
    };
    return Object.entries(replacements).reduce((html, [token, value]) => html.replaceAll(token, value), template);
}

function escapeHTML(value) {
    return value.replaceAll('&', '&amp;').replaceAll('"', '&quot;').replaceAll('<', '&lt;').replaceAll('>', '&gt;');
}
