import {useEffect, useRef} from 'react';
import {ChevronLeft, ChevronRight, Maximize2, X} from 'lucide-react';
import scenarioAnalytics from './assets/scenarios/scenario-analytics.png';
import scenarioKnowledge from './assets/scenarios/scenario-knowledge.png';
import scenarioOperations from './assets/scenarios/scenario-operations.png';
import scenarioSales from './assets/scenarios/scenario-sales.png';
import scenarioService from './assets/scenarios/scenario-service.png';

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

export const scenarios: Scenario[] = [
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

export function ScenarioCard({scenario, onOpen}: {scenario: Scenario; onOpen: () => void}) {
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

export function ScenarioDialog({activeIndex, onActiveIndexChange, onClose}: {
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
