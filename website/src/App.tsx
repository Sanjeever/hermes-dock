import {type FormEvent, useCallback, useEffect, useRef, useState} from 'react';
import {ArrowDown, ArrowRight, Check, ChevronLeft, ChevronRight, Clock3, FileText, LockKeyhole, Maximize2, MessageCircleMore, TableProperties, X} from 'lucide-react';
import deploymentVisual from './assets/deployment-local-options.jpg';
import logo from './assets/qizhih-box-logo.png';
import scenarioAnalytics from './assets/scenarios/scenario-analytics.png';
import scenarioKnowledge from './assets/scenarios/scenario-knowledge.png';
import scenarioOperations from './assets/scenarios/scenario-operations.png';
import scenarioSales from './assets/scenarios/scenario-sales.png';
import scenarioService from './assets/scenarios/scenario-service.png';

const leadEndpoint = '/api/demo-requests';
type FormField = 'name' | 'company' | 'phone';
type FormState = 'idle' | 'submitting' | 'success' | 'error';
type Scenario = {
    number: string;
    label: string;
    title: string;
    description: string;
    resultLabel: string;
    result: string;
    tone: string;
    image: string;
    imageAlt: string;
};

const navigation = [
    ['能力', '#capabilities'],
    ['场景', '#scenarios'],
    ['本地部署', '#deployment'],
    ['操作手册', '/manual/'],
];

const capabilities = [
    {
        icon: Clock3,
        title: '到点就做，不必反复提醒',
        copy: '每天早报、每周复盘、逾期提醒。把需要记住的工作，交给会主动推进的助手。',
        className: 'capability-primary',
    },
    {
        icon: MessageCircleMore,
        title: '就在你已经在用的地方',
        copy: '接入微信、企业微信和飞书。无需改变团队的沟通习惯。',
        className: 'capability-channel',
    },
    {
        icon: FileText,
        title: '资料会被真正用起来',
        copy: '理解制度、产品资料与常见问答，给出贴合业务的答案和初稿。',
        className: 'capability-knowledge',
    },
    {
        icon: TableProperties,
        title: '一堆表格，也能交付一份答案',
        copy: '合并多份 Excel 明细，统一口径、核对异常，并生成可继续使用的分析报表。',
        className: 'capability-analytics',
    },
    {
        icon: LockKeyhole,
        title: '部署在企业指定设备',
        copy: '模型服务由企业选择，业务资料留在企业掌控范围内。',
        className: 'capability-security',
    },
];

const scenarios: Scenario[] = [
    {
        number: '01',
        label: 'SALES',
        title: '销售跟进助手',
        description: '记录商机、提醒跟进、生成个性化触达，让每一个机会都有回应。',
        resultLabel: '客户实测',
        result: '新客跟进率从 30% 提升到 75%',
        tone: 'sales',
        image: scenarioSales,
        imageAlt: '飞书对话演示：销售助手读取商机台账，整理当天必须优先跟进的三位客户',
    },
    {
        number: '02',
        label: 'SERVICE',
        title: '客户咨询助手',
        description: '从产品资料中找到答案，快速形成有依据、有分寸的回复。',
        resultLabel: '客户实测',
        result: '客服平均响应时间缩短约 40%',
        tone: 'service',
        image: scenarioService,
        imageAlt: '飞书对话演示：客户咨询助手依据部署与数据资料，生成可直接发送的客户回复',
    },
    {
        number: '03',
        label: 'KNOWLEDGE',
        title: '内部资料助手',
        description: '制度、培训、流程不再散落在文件夹和某个人的记忆里。',
        resultLabel: '工作变化',
        result: '少问人，少翻文件，答案随问随得',
        tone: 'knowledge',
        image: scenarioKnowledge,
        imageAlt: '飞书对话演示：内部资料助手对比客户资料库和报销权限的办理流程',
    },
    {
        number: '04',
        label: 'OPERATIONS',
        title: '运营与项目助手',
        description: '推送数据早报、识别异常、整理待办，让团队始终知道下一步。',
        resultLabel: '工作变化',
        result: '让进展清晰，让行动不脱轨',
        tone: 'operations',
        image: scenarioOperations,
        imageAlt: '飞书对话演示：运营助手汇总昨日表现，识别异常并给出当天行动建议',
    },
    {
        number: '05',
        label: 'ANALYTICS',
        title: '经营数据分析助手',
        description: '交叉分析多份 Excel，核对口径与异常，直接交付可继续使用的经营报表。',
        resultLabel: '工作变化',
        result: '一次完成跨表分析、异常核对与报表交付',
        tone: 'analytics',
        image: scenarioAnalytics,
        imageAlt: '飞书对话演示：经营数据分析助手读取多份 Excel，检查关联问题并生成分析报表',
    },
];

function ScenarioCard({scenario, onOpen}: {scenario: Scenario; onOpen: () => void}) {
    return (
        <article className={`scenario-card scenario-${scenario.tone}`}>
            <header className="scenario-card-header">
                <p><span>{scenario.number}</span><span>{scenario.label}</span></p>
                <h3>{scenario.title}</h3>
            </header>
            <figure className="scenario-figure">
                <button aria-label={`查看${scenario.title}的完整飞书对话`} className="scenario-preview" onClick={onOpen} type="button">
                    <img alt={scenario.imageAlt} decoding="async" height={961} loading="lazy" src={scenario.image} width={1325} />
                    <span className="scenario-expand">查看完整对话 <Maximize2 aria-hidden="true" size={15} /></span>
                </button>
            </figure>
            <footer className="scenario-card-footer">
                <p>{scenario.description}</p>
                <div className="scenario-result"><span>{scenario.resultLabel}</span><strong>{scenario.result}</strong></div>
            </footer>
        </article>
    );
}

function ScenarioDialog({activeIndex, onActiveIndexChange, onClose}: {
    activeIndex: number;
    onActiveIndexChange: (index: number) => void;
    onClose: () => void;
}) {
    const dialogRef = useRef<HTMLDialogElement>(null);
    const mediaRef = useRef<HTMLDivElement>(null);
    const triggerRef = useRef<HTMLElement | null>(null);
    const scenario = scenarios[activeIndex];

    useEffect(() => {
        const dialog = dialogRef.current;
        if (!dialog) return;

        triggerRef.current = document.activeElement as HTMLElement | null;
        const previousOverflow = document.body.style.overflow;
        document.body.style.overflow = 'hidden';
        dialog.showModal();

        return () => {
            if (dialog.open) dialog.close();
            document.body.style.overflow = previousOverflow;
            triggerRef.current?.focus();
        };
    }, [onClose]);

    useEffect(() => {
        mediaRef.current?.scrollTo({left: 0, top: 0});
    }, [activeIndex]);

    const showPrevious = () => onActiveIndexChange((activeIndex - 1 + scenarios.length) % scenarios.length);
    const showNext = () => onActiveIndexChange((activeIndex + 1) % scenarios.length);

    return (
        <dialog
            aria-labelledby="scenario-dialog-title"
            className="scenario-dialog"
            onCancel={(event) => { event.preventDefault(); onClose(); }}
            onClick={(event) => { if (event.target === dialogRef.current) onClose(); }}
            onKeyDown={(event) => {
                if (event.key === 'ArrowLeft') { event.preventDefault(); showPrevious(); }
                if (event.key === 'ArrowRight') { event.preventDefault(); showNext(); }
            }}
            ref={dialogRef}
        >
            <div className="scenario-dialog-shell">
                <header className="scenario-dialog-header">
                    <div>
                        <span>{scenario.number} / {String(scenarios.length).padStart(2, '0')}</span>
                        <h2 id="scenario-dialog-title">{scenario.title}</h2>
                    </div>
                    <button aria-label="关闭完整对话" className="scenario-dialog-close" onClick={onClose} type="button"><X aria-hidden="true" size={21} /></button>
                </header>
                <div className="scenario-dialog-media" ref={mediaRef}>
                    <img alt={scenario.imageAlt} className="scenario-dialog-image" height={961} src={scenario.image} width={1325} />
                </div>
                <p className="scenario-dialog-hint">移动端可左右滑动查看对话细节</p>
                <nav aria-label="切换场景" className="scenario-dialog-nav">
                    <button aria-label="查看上一个场景" onClick={showPrevious} type="button"><ChevronLeft aria-hidden="true" size={21} /><span>上一个</span></button>
                    <button aria-label="查看下一个场景" onClick={showNext} type="button"><span>下一个</span><ChevronRight aria-hidden="true" size={21} /></button>
                </nav>
            </div>
        </dialog>
    );
}

function Brand({inverse = false}: {inverse?: boolean}) {
    return (
        <a className={`brand ${inverse ? 'brand-inverse' : ''}`} href="#top" aria-label="企智盒首页">
            <img src={logo} alt="企智盒 logo" />
            <span>企智盒</span>
        </a>
    );
}

function ArrowLink({children, href = '#contact'}: {children: string; href?: string}) {
    return <a className="arrow-link" href={href}>{children}<ArrowRight aria-hidden="true" size={17} /></a>;
}

function ProductDevice({className = ''}: {className?: string}) {
    return (
        <div className={`product-device ${className}`} aria-hidden="true">
            <div className="product-device-floor" />
            <div className="product-device-machine">
                <div className="product-device-top-surface">
                    <div className="product-device-inlay"><img src={logo} alt="" /></div>
                </div>
                <div className="product-device-front">
                    <div className="product-device-ports"><i /><i /><i /></div>
                    <span className="product-device-led" />
                </div>
            </div>
        </div>
    );
}

function App() {
    const scope = useRef<HTMLDivElement>(null);
    const [formState, setFormState] = useState<FormState>('idle');
    const [formMessage, setFormMessage] = useState('');
    const [fieldErrors, setFieldErrors] = useState<Partial<Record<FormField, string>>>({});
    const [activeScenarioView, setActiveScenarioView] = useState(0);
    const [openScenarioIndex, setOpenScenarioIndex] = useState<number | null>(null);
    const closeScenarioDialog = useCallback(() => setOpenScenarioIndex(null), []);

    useEffect(() => {
        if (window.matchMedia('(prefers-reduced-motion: reduce)').matches) return;

        let cancelled = false;
        let cleanup: (() => void) | undefined;

        const setupAnimations = async () => {
            const [{gsap}, {ScrollTrigger}] = await Promise.all([
                import('gsap'),
                import('gsap/ScrollTrigger'),
            ]);
            if (cancelled) return;

            gsap.registerPlugin(ScrollTrigger);
            ScrollTrigger.config({ignoreMobileResize: true, limitCallbacks: true});
            const media = gsap.matchMedia();
            const context = gsap.context(() => {
                const wordNodes = gsap.utils.toArray<HTMLElement>('.reveal-word');
                gsap.to(wordNodes, {
                    opacity: 1,
                    y: 0,
                    stagger: 0.018,
                    duration: 0.48,
                    ease: 'power2.out',
                    scrollTrigger: {
                        trigger: '.statement',
                        start: 'top 68%',
                        toggleActions: 'play none none reverse',
                    },
                });

                media.add('(min-width: 761px)', () => {
                    const cards = gsap.utils.toArray<HTMLElement>('.scenario-card');
                    if (!cards.length) return;

                    cards.forEach((card, index) => {
                        gsap.fromTo(card, {y: 42, opacity: 0.18}, {
                            y: 0,
                            opacity: 1,
                            duration: 0.62,
                            ease: 'power3.out',
                            force3D: true,
                            scrollTrigger: {
                                trigger: card,
                                start: 'top 78%',
                                toggleActions: 'play none none reverse',
                            },
                        });
                        ScrollTrigger.create({
                            trigger: card,
                            start: 'top 52%',
                            end: 'bottom 52%',
                            onEnter: () => setActiveScenarioView(index),
                            onEnterBack: () => setActiveScenarioView(index),
                        });
                    });
                });

                gsap.utils.toArray<HTMLElement>('.scale-in').forEach((element) => {
                    gsap.fromTo(element, {y: 28, opacity: 0}, {
                        y: 0,
                        opacity: 1,
                        duration: 0.7,
                        ease: 'power3.out',
                        force3D: true,
                        scrollTrigger: {
                            trigger: element,
                            start: 'top 82%',
                            toggleActions: 'play none none reverse',
                        },
                    });
                });
            }, scope);

            cleanup = () => {
                media.revert();
                context.revert();
            };
        };

        void setupAnimations();

        return () => {
            cancelled = true;
            cleanup?.();
        };
    }, []);

    const clearFieldError = (field: FormField) => () => {
        setFieldErrors((current) => ({...current, [field]: undefined}));
    };

    const handleSubmit = async (event: FormEvent<HTMLFormElement>) => {
        event.preventDefault();
        const form = event.currentTarget;
        const values = new FormData(form);
        const name = String(values.get('name') || '').trim();
        const company = String(values.get('company') || '').trim();
        const phone = String(values.get('phone') || '').trim();
        const errors: Partial<Record<FormField, string>> = {};

        if (name.length < 2) errors.name = '请填写至少两个字的称呼。';
        if (company.length < 2) errors.company = '请填写企业名称。';
        if (!/^[0-9+()\-\s]{7,20}$/.test(phone)) errors.phone = '请填写有效的联系电话。';

        setFieldErrors(errors);
        if (Object.keys(errors).length) {
            setFormState('error');
            setFormMessage('请检查标出的信息后再提交。');
            return;
        }

        setFormState('submitting');
        setFormMessage('正在提交预约信息…');
        let failureMessage = '暂时无法提交预约，请稍后重试或直接联系企智盒顾问。';
        try {
            const response = await fetch(leadEndpoint, {
                method: 'POST',
                headers: {'Content-Type': 'application/json'},
                body: JSON.stringify(Object.fromEntries(values)),
            });
            if (!response.ok) {
                if (response.status === 429) failureMessage = '提交过于频繁，请稍后再试。';
                else if (response.status >= 500) failureMessage = '预约服务暂时不可用，请稍后重试。';
                else failureMessage = '预约信息有误，请检查后重新提交。';
                throw new Error('预约请求失败');
            }
            form.reset();
            setFormState('success');
            setFormMessage('已收到预约。企智盒顾问会在 1 个工作日内与您联系。');
        } catch {
            setFormState('error');
            setFormMessage(failureMessage);
        }
    };

    return (
        <main ref={scope} id="top" className="site-shell">
            <header className="site-header">
                <Brand />
                <nav aria-label="主导航">
                    {navigation.map(([label, href]) => <a key={href} href={href}>{label}</a>)}
                </nav>
                <a className="header-cta" href="#contact">预约演示 <ArrowRight aria-hidden="true" size={16} /></a>
            </header>

            <section className="hero" aria-labelledby="hero-title">
                <div className="hero-halo" />
                <p className="hero-kicker hero-reveal">给每个明确岗位，一个长期协作的 AI 助手</p>
                <h1 id="hero-title" className="hero-reveal">
                    把 AI 从问一问<br />
                    <span>变成</span>帮我干<span className="inline-logo"><img src={logo} alt="" /></span>
                </h1>
                <p className="hero-copy hero-reveal">企智盒把 AI 放进企业日常工作：理解资料、长期记忆、主动执行，并在团队熟悉的沟通工具中持续协作。</p>
                <div className="hero-actions hero-reveal">
                    <a className="button button-dark" href="#contact">预约演示 <ArrowRight aria-hidden="true" size={18} /></a>
                    <a className="button button-light" href="#capabilities">了解企智盒 <ArrowDown aria-hidden="true" size={18} /></a>
                </div>
                <ProductDevice className="hero-product hero-reveal" />
            </section>

            <section className="statement" aria-label="产品理念">
                <p>{'不是再打开一个工具，而是让团队在每一天的工作里，多一位懂业务、会推进、记得住的协作者。'.split('').map((character, index) => <span className="reveal-word" key={`${character}-${index}`}>{character}</span>)}</p>
            </section>

            <section id="capabilities" className="capabilities section-wrap" aria-labelledby="capabilities-title">
                <div className="section-intro">
                    <h2 id="capabilities-title">从一句指令开始，<br />让工作自己往前走。</h2>
                </div>
                <div className="capability-grid">
                    {capabilities.map(({icon: Icon, title, copy, className}) => (
                        <article key={title} className={`capability-card ${className} scale-in`}>
                            <div className="capability-icon"><Icon aria-hidden="true" size={23} strokeWidth={1.7} /></div>
                            <div><h3>{title}</h3><p>{copy}</p></div>
                        </article>
                    ))}
                </div>
            </section>

            <section id="scenarios" className="story-stage section-wrap" aria-labelledby="scenarios-title">
                <div className="story-copy">
                    <h2 id="scenarios-title">每一个助手，<br />都有一份明确的工作。</h2>
                    <p className="story-copy-detail">先解决一个最常发生、最耗时间的问题。看到效果后，再逐步增加更多助手。</p>
                    <div aria-hidden="true" className="story-progress">
                        <span>正在浏览</span>
                        <strong>{scenarios[activeScenarioView].number} / {String(scenarios.length).padStart(2, '0')}</strong>
                        <p>{scenarios[activeScenarioView].title}</p>
                        <div>{scenarios.map((scenario, index) => <i className={index === activeScenarioView ? 'is-active' : ''} key={scenario.number} />)}</div>
                    </div>
                    <ArrowLink href="#contact">预约演示</ArrowLink>
                </div>
                <div className="scenario-stack">
                    {scenarios.map((scenario, index) => <ScenarioCard key={scenario.number} onOpen={() => setOpenScenarioIndex(index)} scenario={scenario} />)}
                </div>
            </section>

            <section id="deployment" className="deployment section-wrap" aria-labelledby="deployment-title">
                <div className="deployment-art scale-in">
                    <img className="deployment-visual" src={deploymentVisual} alt="一体机与企业服务器机房" loading="lazy" />
                </div>
                <div className="deployment-copy">
                    <p>数据留在掌控之中</p>
                    <h2 id="deployment-title">AI 的能力，<br />部署在你的业务现场。</h2>
                    <p className="deployment-detail">可选择预装环境的一体机，也可部署在企业现有的合适设备上。基础部署、平台接入和培训，都由企智盒团队陪你完成。</p>
                    <ul>
                        <li><Check aria-hidden="true" size={18} /> 企业自行选择模型服务</li>
                        <li><Check aria-hidden="true" size={18} /> 资料与业务规则由企业掌控</li>
                        <li><Check aria-hidden="true" size={18} /> 不需要专职 AI 或 IT 团队</li>
                    </ul>
                </div>
            </section>

            <section className="marquee" aria-label="企智盒可协作的平台">
                <div className="marquee-track"><span>微信</span><i>·</i><span>企业微信</span><i>·</i><span>飞书</span><i>·</i><span>长期记忆</span><i>·</i><span>主动执行</span><i>·</i><span>本地使用</span><i>·</i><span>微信</span><i>·</i><span>企业微信</span><i>·</i><span>飞书</span><i>·</i><span>长期记忆</span><i>·</i></div>
            </section>

            <section id="contact" className="contact section-wrap" aria-labelledby="contact-title">
                <div className="contact-intro">
                    <h2 id="contact-title">看看企智盒，<br />能先帮你的团队做什么。</h2>
                    <p>适合有稳定业务、希望从销售、客服、运营或内部协作开始的团队。预约演示后，可根据你的实际情况安排上门评估。</p>
                </div>
                <form className="contact-form" noValidate onSubmit={handleSubmit}>
                    <div className="form-trap" aria-hidden="true">
                        <label htmlFor="website">网站</label>
                        <input id="website" name="website" type="text" tabIndex={-1} autoComplete="off" />
                    </div>
                    <div className="form-field">
                        <label htmlFor="name">姓名</label>
                        <input aria-describedby={fieldErrors.name ? 'name-error' : undefined} aria-invalid={Boolean(fieldErrors.name)} id="name" name="name" placeholder="如何称呼您" autoComplete="name" maxLength={50} onChange={clearFieldError('name')} />
                        {fieldErrors.name && <p className="field-error" id="name-error">{fieldErrors.name}</p>}
                    </div>
                    <div className="form-field">
                        <label htmlFor="company">企业名称</label>
                        <input aria-describedby={fieldErrors.company ? 'company-error' : undefined} aria-invalid={Boolean(fieldErrors.company)} id="company" name="company" placeholder="您的企业名称" autoComplete="organization" maxLength={100} onChange={clearFieldError('company')} />
                        {fieldErrors.company && <p className="field-error" id="company-error">{fieldErrors.company}</p>}
                    </div>
                    <div className="form-field">
                        <label htmlFor="phone">联系电话</label>
                        <input aria-describedby={fieldErrors.phone ? 'phone-error' : undefined} aria-invalid={Boolean(fieldErrors.phone)} id="phone" name="phone" placeholder="方便联系您的电话" autoComplete="tel" type="tel" inputMode="tel" maxLength={20} onChange={clearFieldError('phone')} />
                        {fieldErrors.phone && <p className="field-error" id="phone-error">{fieldErrors.phone}</p>}
                    </div>
                    <div className="form-field">
                        <label htmlFor="need">想先解决什么问题</label>
                        <textarea id="need" name="need" placeholder="例如：客户跟进、资料查询、运营日报……" rows={3} maxLength={1000} />
                    </div>
                    <p className="form-privacy">提交即表示你同意企智盒仅为安排本次演示与评估联系你。</p>
                    <button aria-busy={formState === 'submitting'} className="button button-coral" disabled={formState === 'submitting'} type="submit">{formState === 'submitting' ? '正在提交' : '预约演示'} <ArrowRight aria-hidden="true" size={18} /></button>
                    {formState !== 'idle' && <p className={`form-status form-status-${formState}`} role={formState === 'error' ? 'alert' : 'status'}>{formMessage}</p>}
                </form>
            </section>

            <footer className="site-footer">
                <Brand inverse />
                <div className="footer-meta">
                    <p className="footer-tagline">把 AI 放进企业日常工作。</p>
                    <div className="footer-registration">
                        <a href="/manual/">操作手册</a>
                        <span>广西尚企云链科技有限公司</span>
                        <a href="https://beian.miit.gov.cn/" target="_blank" rel="noreferrer">桂ICP备2024050395号-3</a>
                    </div>
                </div>
                <a className="footer-return" href="#top">回到顶部 <ArrowRight aria-hidden="true" size={15} /></a>
            </footer>
            {openScenarioIndex !== null && <ScenarioDialog activeIndex={openScenarioIndex} onActiveIndexChange={setOpenScenarioIndex} onClose={closeScenarioDialog} />}
        </main>
    );
}

export default App;
