import {isValidElement, useEffect, useRef, useState, type ReactNode} from 'react';
import {ArrowDown, ArrowLeft, ArrowUpRight, Menu, Printer, X} from 'lucide-react';
import Markdown, {type Components} from 'react-markdown';
import remarkGfm from 'remark-gfm';
import manualSource from '../content/manual.md?raw';
import logo from './assets/qizhih-box-logo.png';

type ManualMeta = {
    title: string;
    description: string;
    updated: string;
};

type TocItem = {
    id: string;
    label: string;
};

function parseManual(source: string): {meta: ManualMeta; content: string} {
    const match = source.match(/^---\n([\s\S]*?)\n---\n/);
    if (!match) throw new Error('操作手册缺少 front matter');

    const values = Object.fromEntries(match[1].split('\n').map((line) => {
        const separator = line.indexOf(':');
        return [line.slice(0, separator).trim(), line.slice(separator + 1).trim()];
    }));

    return {
        meta: {
            title: values.title,
            description: values.description,
            updated: values.updated,
        },
        content: source.slice(match[0].length),
    };
}

function headingText(children: ReactNode): string {
    if (Array.isArray(children)) return children.map(headingText).join('');
    if (typeof children === 'string' || typeof children === 'number') return String(children);
    if (isValidElement<{children?: ReactNode}>(children)) return headingText(children.props.children);
    return '';
}

function headingID(value: string): string {
    return value
        .toLocaleLowerCase('zh-CN')
        .replace(/[`*_~]/g, '')
        .replace(/[^\p{Letter}\p{Number}\s-]/gu, '')
        .trim()
        .replace(/\s+/g, '-');
}

function plainHeading(value: string): string {
    return value.replace(/[`*_~]/g, '').replace(/\[(.*?)\]\(.*?\)/g, '$1').trim();
}

function displayDate(value: string): string {
    const match = value.match(/^(\d{4})-(\d{2})-(\d{2})$/);
    if (!match) return value;
    return `${match[1]}年${Number(match[2])}月${Number(match[3])}日`;
}

const manual = parseManual(manualSource);
const toc = manual.content.split('\n').flatMap<TocItem>((line) => {
    const match = line.match(/^(##)\s+(.+)$/);
    if (!match) return [];
    const label = plainHeading(match[2]);
    return [{id: headingID(label), label}];
});

const markdownComponents: Components = {
    h2: ({children}) => {
        const label = headingText(children);
        const id = headingID(label);
        return <h2 id={id}><a href={`#${id}`}>{children}</a></h2>;
    },
    h3: ({children}) => {
        const label = headingText(children);
        const id = headingID(label);
        return <h3 id={id}><a href={`#${id}`}>{children}</a></h3>;
    },
    a: ({href = '', children}) => {
        const external = /^https?:\/\//.test(href);
        return (
            <a href={href} target={external ? '_blank' : undefined} rel={external ? 'noreferrer' : undefined}>
                {children}{external && <ArrowUpRight aria-hidden="true" size={13} />}
            </a>
        );
    },
    table: ({children}) => <div className="manual-table-wrap"><table>{children}</table></div>,
    img: ({src, alt}) => <img src={src} alt={alt || ''} loading="lazy" />,
};

function ManualPage() {
    const [activeHeading, setActiveHeading] = useState(toc[0]?.id || '');
    const [tocOpen, setTocOpen] = useState(false);
    const menuButtonRef = useRef<HTMLButtonElement>(null);
    const mobileTocRef = useRef<HTMLDivElement>(null);

    useEffect(() => {
        const headings = Array.from(document.querySelectorAll<HTMLElement>('.manual-article h2'));
        const observer = new IntersectionObserver((entries) => {
            for (const entry of entries) {
                if (entry.isIntersecting) setActiveHeading(entry.target.id);
            }
        }, {rootMargin: '0px 0px -72%'});
        headings.forEach((heading) => observer.observe(heading));
        return () => observer.disconnect();
    }, []);

    useEffect(() => {
        if (!tocOpen) return;
        const panel = mobileTocRef.current;
        const focusable = Array.from(panel?.querySelectorAll<HTMLElement>('a[href], button:not([disabled])') || []);
        const previousOverflow = document.body.style.overflow;
        document.body.style.overflow = 'hidden';
        focusable[0]?.focus();
        const handleKeyDown = (event: KeyboardEvent) => {
            if (event.key === 'Escape') {
                setTocOpen(false);
                return;
            }
            if (event.key !== 'Tab' || focusable.length === 0) return;
            const first = focusable[0];
            const last = focusable[focusable.length - 1];
            if (event.shiftKey && document.activeElement === first) {
                event.preventDefault();
                last.focus();
            } else if (!event.shiftKey && document.activeElement === last) {
                event.preventDefault();
                first.focus();
            }
        };
        window.addEventListener('keydown', handleKeyDown);
        return () => {
            window.removeEventListener('keydown', handleKeyDown);
            document.body.style.overflow = previousOverflow;
            menuButtonRef.current?.focus();
        };
    }, [tocOpen]);

    useEffect(() => {
        const mobileLayout = window.matchMedia('(max-width: 760px)');
        const closeOnDesktop = (event: MediaQueryListEvent) => {
            if (!event.matches) setTocOpen(false);
        };
        mobileLayout.addEventListener('change', closeOnDesktop);
        return () => mobileLayout.removeEventListener('change', closeOnDesktop);
    }, []);

    return (
        <div className="manual-shell">
            <a className="manual-skip-link" href="#manual-content">跳到手册正文</a>
            <header className="manual-header">
                <a className="manual-brand" href="/" aria-label="返回企智盒官网">
                    <img src={logo} alt="" />
                    <span>企智盒</span>
                </a>
                <div className="manual-header-actions">
                    <a className="manual-back" href="/"><ArrowLeft aria-hidden="true" size={16} /> 官网首页</a>
                    <button className="manual-print" type="button" onClick={() => window.print()}><Printer aria-hidden="true" size={16} /> 打印</button>
                    <button ref={menuButtonRef} className="manual-menu" type="button" aria-expanded={tocOpen} aria-controls="manual-mobile-toc" onClick={() => setTocOpen((open) => !open)}>
                        {tocOpen ? <X aria-hidden="true" size={18} /> : <Menu aria-hidden="true" size={18} />}
                        目录
                    </button>
                </div>
            </header>

            <section className="manual-hero" aria-labelledby="manual-title">
                <div className="manual-hero-halo" aria-hidden="true" />
                <div className="manual-hero-content">
                    <p className="manual-kicker">从第一次打开，到助手开始工作</p>
                    <h1 id="manual-title"><span>{manual.meta.title.slice(0, 3)}</span><br />{manual.meta.title.slice(3)}</h1>
                    <p className="manual-description">{manual.meta.description}</p>
                    <div className="manual-hero-actions">
                        <a className="manual-primary-action" href="#快速上手">开始阅读 <ArrowDown aria-hidden="true" size={17} /></a>
                        <button className="manual-secondary-action" type="button" onClick={() => window.print()}><Printer aria-hidden="true" size={16} /> 打印手册</button>
                    </div>
                    <div className="manual-meta">
                        <span>最后更新：{displayDate(manual.meta.updated)}</span>
                        <span>首次配置 · 日常管理 · 故障排查</span>
                    </div>
                </div>
            </section>

            {tocOpen && <button className="manual-toc-backdrop" type="button" aria-label="关闭目录" onClick={() => setTocOpen(false)} />}
            <div ref={mobileTocRef} className={`manual-mobile-toc ${tocOpen ? 'open' : ''}`} id="manual-mobile-toc" role="dialog" aria-modal="true" aria-label="操作手册目录" hidden={!tocOpen}>
                <div className="manual-mobile-toc-heading">
                    <strong>操作手册目录</strong>
                    <button type="button" aria-label="关闭目录" onClick={() => setTocOpen(false)}><X aria-hidden="true" size={18} /></button>
                </div>
                <TocList activeHeading={activeHeading} onSelect={() => setTocOpen(false)} />
            </div>

            <div className="manual-layout">
                <aside className="manual-toc" aria-label="操作手册目录">
                    <div>
                        <p>操作手册</p>
                        <TocList activeHeading={activeHeading} />
                        <span className="manual-toc-updated">更新于 {displayDate(manual.meta.updated)}</span>
                    </div>
                </aside>
                <main className="manual-article" id="manual-content">
                    <Markdown remarkPlugins={[remarkGfm]} components={markdownComponents}>{manual.content}</Markdown>
                </main>
            </div>

            <footer className="manual-footer">
                <a className="manual-brand manual-brand-inverse" href="/" aria-label="返回企智盒官网">
                    <img src={logo} alt="" />
                    <span>企智盒</span>
                </a>
                <div className="manual-footer-meta">
                    <p>把 AI 放进企业日常工作。</p>
                    <div><span>广西尚企云链科技有限公司</span><a href="https://beian.miit.gov.cn/" target="_blank" rel="noreferrer">桂ICP备2024050395号-3</a></div>
                </div>
                <a href="#manual-title">回到顶部</a>
            </footer>
        </div>
    );
}

function TocList({activeHeading, onSelect}: {activeHeading: string; onSelect?: () => void}) {
    return (
        <ol>
            {toc.map((item) => (
                <li key={item.id} className={activeHeading === item.id ? 'active' : ''}>
                    <a href={`#${item.id}`} aria-current={activeHeading === item.id ? 'location' : undefined} onClick={onSelect}>{item.label}</a>
                </li>
            ))}
        </ol>
    );
}

export default ManualPage;
