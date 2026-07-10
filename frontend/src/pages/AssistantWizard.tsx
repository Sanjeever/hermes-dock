import {CheckCircle2, ChevronLeft, ChevronRight, RefreshCcw} from 'lucide-react';
import {PlatformsPage} from './PlatformsPage';
import {SoulPage} from './SoulPage';
import {ModelServiceStep} from './ModelServiceStep';
import type {EnvVar, ModelConfig, ModelOption, PlatformKey, ProviderConfig, WizardStep} from '../types';
import {wizardStepHelp} from './assistantUtils';

export function AssistantWizard(props: {
    step: WizardStep;
    setupDone: boolean;
    profileID: string;
    profileName: string;
    env: EnvVar[];
    providers: ProviderConfig;
    setProviders: (value: ProviderConfig) => void;
    selectedProvider: string;
    setSelectedProvider: (value: string) => void;
    model: ModelConfig | null;
    setModel: (value: ModelConfig) => void;
    modelOptions: ModelOption[];
    modelListStatus: string;
    modelTestStatus: string;
    busy: boolean;
    showApiKey: boolean;
    setShowApiKey: (value: boolean) => void;
    soulContent: string;
    setSoulContent: (value: string) => void;
    soulStatus: string;
    soulDirty: boolean;
    setSoulDirty: (value: boolean) => void;
    qrData: string;
    qrStatus: string;
    modelDirty: boolean;
    platformDirty: boolean;
    selectedPlatform: PlatformKey;
    setSelectedPlatform: (value: PlatformKey) => void;
    setEnv: (value: EnvVar[]) => void;
    hasPlatformBinding: boolean;
    onStep: (step: WizardStep | null) => void;
    onSaveModelService: () => Promise<boolean>;
    onFetchModels: () => void;
    onTestModel: () => void;
    onSaveSoul: () => Promise<boolean>;
    onDiscardSoul: () => void;
    onRestoreDefaultSoul: () => Promise<boolean>;
    onWeixinLogin: () => void;
    onCancelWeixin: () => void;
    onSaveWeCom: () => Promise<boolean>;
    onSaveFeishu: () => Promise<boolean>;
    onUnbindPlatform: (platform: PlatformKey) => void;
    onSaveCurrentPlatform: () => Promise<boolean>;
    onFinishSetup: (apply: boolean) => Promise<boolean>;
    onOpenProviders: () => void;
}) {
    const steps: Array<{ id: WizardStep; label: string }> = [
        {id: 'model', label: '模型服务'},
        {id: 'soul', label: '人格设定'},
        {id: 'platforms', label: '平台绑定'},
        {id: 'finish', label: '完成'},
    ];
    const index = steps.findIndex((item) => item.id === props.step);
    const previous = index > 0 ? steps[index - 1].id : null;
    const next = index < steps.length - 1 ? steps[index + 1].id : null;
    const modelReady = !!props.model?.provider && !!props.model?.default?.trim();
    const goToStep = async (step: WizardStep) => {
        if (step === props.step) return;
        if (props.modelDirty && !(await props.onSaveModelService())) return;
        if (props.soulDirty && !(await props.onSaveSoul())) return;
        if (props.platformDirty && !(await props.onSaveCurrentPlatform())) return;
        props.onStep(step);
    };

    return (
        <div className="wizard-stack">
            <div className="wizard-steps">
                {steps.map((item, itemIndex) => (
                    <button key={item.id} className={props.step === item.id ? 'active' : itemIndex < index ? 'done' : ''} onClick={() => goToStep(item.id)} title={item.label} aria-label={item.label} disabled={!props.setupDone || props.busy}>
                        <span>{itemIndex + 1}</span>
                        <em>{item.label}</em>
                    </button>
                ))}
            </div>
            {props.step === 'model' && props.model && (
                <ModelServiceStep
                    providers={props.providers}
                    setProviders={props.setProviders}
                    selectedProvider={props.selectedProvider}
                    setSelectedProvider={props.setSelectedProvider}
                    model={props.model}
                    setModel={props.setModel}
                    modelOptions={props.modelOptions}
                    modelListStatus={props.modelListStatus}
                    modelTestStatus={props.modelTestStatus}
                    modelDirty={props.modelDirty}
                    busy={props.busy}
                    showApiKey={props.showApiKey}
                    setShowApiKey={props.setShowApiKey}
                    onFetchModels={props.onFetchModels}
                    onTestModel={props.onTestModel}
                    onSaveModelService={props.onSaveModelService}
                    stepLabel={`第 ${index + 1} 步，共 ${steps.length} 步`}
                    stepHelp={wizardStepHelp(props.step)}
                    onNext={() => props.onStep('soul')}
                    onOpenProviders={props.onOpenProviders}
                />
            )}
            {props.step === 'soul' && (
                <div className="wizard-panel">
                    <SoulPage
                        profileID={props.profileID}
                        content={props.soulContent}
                        setContent={(value) => {
                            props.setSoulContent(value);
                            props.setSoulDirty(true);
                        }}
                        status={props.soulStatus}
                        dirty={props.soulDirty}
                        busy={props.busy}
                        onSave={props.onSaveSoul}
                        onDiscard={props.onDiscardSoul}
                        onRestoreDefault={props.onRestoreDefaultSoul}
                    />
                    <WizardNav previous={previous} next={next} busy={props.busy} onPrevious={(step) => step && goToStep(step)} onNext={async () => {
                        if (props.soulDirty && !(await props.onSaveSoul())) return;
                        props.onStep('platforms');
                    }} nextLabel="保存并下一步"/>
                </div>
            )}
            {props.step === 'platforms' && (
                <div className="wizard-panel">
                    <PlatformsPage
                        env={props.env}
                        setEnv={props.setEnv}
                        qrData={props.qrData}
                        qrStatus={props.qrStatus}
                        selected={props.selectedPlatform}
                        setSelected={props.setSelectedPlatform}
                        busy={props.busy}
                        onWeixinLogin={props.onWeixinLogin}
                        onCancelWeixin={props.onCancelWeixin}
                        onSaveWeCom={props.onSaveWeCom}
                        onSaveFeishu={props.onSaveFeishu}
                        onUnbind={props.onUnbindPlatform}
                    />
                    <WizardNav previous={previous} next={next} busy={props.busy} onPrevious={(step) => step && goToStep(step)} onNext={async () => {
                        if (props.platformDirty && !(await props.onSaveCurrentPlatform())) return;
                        props.onStep('finish');
                    }} nextLabel={props.platformDirty ? '保存并下一步' : props.hasPlatformBinding ? '下一步' : '暂不绑定平台，下一步'}/>
                </div>
            )}
            {props.step === 'finish' && (
                <div className="panel finish-panel">
                    <p className="eyebrow">完成配置</p>
                    <h2>{props.profileName} 的基础配置</h2>
                    <div className="finish-checks">
                        <CheckItem ok={modelReady} label={modelReady ? '已选择模型服务和主模型' : '请先选择模型服务和主模型'}/>
                        <CheckItem ok={true} label="人格设定已可使用"/>
                        <CheckItem ok={props.hasPlatformBinding} label={props.hasPlatformBinding ? '已绑定至少一个平台' : '还没绑定平台，暂时不会接收消息'}/>
                    </div>
                    <div className="actions">
                        <button className="ghost" onClick={() => previous && goToStep(previous)} disabled={props.busy}><ChevronLeft size={16}/>上一步</button>
                        {props.hasPlatformBinding ? (
                            <>
                                <button className="ghost" onClick={() => props.onFinishSetup(false)} disabled={props.busy || !modelReady}>仅完成，稍后应用</button>
                                <button className="primary no-margin" onClick={() => props.onFinishSetup(true)} disabled={props.busy || !modelReady}><RefreshCcw size={16}/>完成并应用配置</button>
                            </>
                        ) : (
                            <button className="primary no-margin" onClick={() => props.onFinishSetup(false)} disabled={props.busy || !modelReady}>稍后绑定平台</button>
                        )}
                    </div>
                </div>
            )}
        </div>
    );
}

function WizardNav(props: { previous: WizardStep | null; next: WizardStep | null; busy: boolean; onPrevious: (step: WizardStep | null) => void; onNext: () => void | Promise<void>; nextLabel?: string }) {
    return (
        <div className="wizard-actions">
            <button className="ghost" onClick={() => props.onPrevious(props.previous)} disabled={props.busy || !props.previous}><ChevronLeft size={16}/>上一步</button>
            <button className="primary no-margin" onClick={props.onNext} disabled={props.busy || !props.next}>{props.nextLabel || '下一步'}<ChevronRight size={16}/></button>
        </div>
    );
}

function CheckItem(props: { ok: boolean; label: string }) {
    return <div className={`check-item ${props.ok ? 'ok-status' : 'warn-status'}`}>{props.ok ? <CheckCircle2 size={16}/> : <RefreshCcw size={16}/>}<span>{props.label}</span></div>;
}
