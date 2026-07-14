import {type FormEvent, useEffect, useRef, useState} from 'react';
import {ArrowDown, ArrowRight, Check, ChevronRight, ClipboardList, Clock3, Database, FileText, LockKeyhole, MessageCircleMore, Send, UsersRound} from 'lucide-react';
import logo from './assets/qizhih-box-logo.png';

const leadEndpoint = '/api/demo-requests';
type FormField = 'name' | 'company' | 'phone';
type FormState = 'idle' | 'submitting' | 'success' | 'error';

const navigation = [
    ['能力', '#capabilities'],
    ['场景', '#scenarios'],
    ['本地部署', '#deployment'],
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
        icon: LockKeyhole,
        title: '部署在企业指定设备',
        copy: '模型服务由企业选择，业务资料留在企业掌控范围内。',
        className: 'capability-security',
    },
];

const scenarios = [
    {
        number: '01',
        title: '销售跟进助手',
        description: '记录商机、提醒跟进、生成个性化触达，让每一个机会都有回应。',
        result: '客户实测：新客跟进率从 30% 提升到 75%',
        tone: 'sales',
    },
    {
        number: '02',
        title: '客户咨询助手',
        description: '从产品资料中找到答案，快速形成有依据、有分寸的回复。',
        result: '客户实测：客服平均响应时间缩短约 40%',
        tone: 'service',
    },
    {
        number: '03',
        title: '内部资料助手',
        description: '制度、培训、流程不再散落在文件夹和某个人的记忆里。',
        result: '少问人，少翻文件，答案随问随得',
        tone: 'knowledge',
    },
    {
        number: '04',
        title: '运营与项目助手',
        description: '推送数据早报、识别异常、整理待办，让团队始终知道下一步。',
        result: '让进展清晰，让行动不脱轨',
        tone: 'operations',
    },
];

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
                    const storyStage = scope.current?.querySelector<HTMLElement>('.story-stage');
                    const storyCopy = scope.current?.querySelector<HTMLElement>('.story-copy');
                    if (!storyStage || !storyCopy || !cards.length) return;

                    ScrollTrigger.create({
                        trigger: storyStage,
                        start: 'top 16%',
                        end: 'bottom bottom',
                        pin: storyCopy,
                        pinSpacing: false,
                        anticipatePin: 1,
                        fastScrollEnd: true,
                        invalidateOnRefresh: true,
                    });

                    cards.forEach((card) => {
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
                    <ArrowLink href="#contact">预约演示</ArrowLink>
                </div>
                <div className="scenario-stack">
                    {scenarios.map((scenario) => (
                        <article key={scenario.number} className={`scenario-card scenario-${scenario.tone}`}>
                            <div className="scenario-visual">
                                <span>{scenario.number}</span>
                                {scenario.tone === 'sales' && <UsersRound aria-hidden="true" size={78} strokeWidth={1.2} />}
                                {scenario.tone === 'service' && <Send aria-hidden="true" size={78} strokeWidth={1.2} />}
                                {scenario.tone === 'knowledge' && <Database aria-hidden="true" size={78} strokeWidth={1.2} />}
                                {scenario.tone === 'operations' && <ClipboardList aria-hidden="true" size={78} strokeWidth={1.2} />}
                            </div>
                            <div className="scenario-content">
                                <h3>{scenario.title}</h3>
                                <p>{scenario.description}</p>
                                <strong>{scenario.result}</strong>
                            </div>
                            <ChevronRight aria-hidden="true" className="scenario-arrow" size={22} />
                        </article>
                    ))}
                </div>
            </section>

            <section id="deployment" className="deployment section-wrap" aria-labelledby="deployment-title">
                <div className="deployment-art scale-in">
                    <ProductDevice className="deployment-product" />
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
                        <span>广西尚企云链科技有限公司</span>
                        <a href="https://beian.miit.gov.cn/" target="_blank" rel="noreferrer">桂ICP备2024050395号-3</a>
                    </div>
                </div>
                <a className="footer-return" href="#top">回到顶部 <ArrowRight aria-hidden="true" size={15} /></a>
            </footer>
        </main>
    );
}

export default App;
