import {useEffect, useState} from 'react';
import {Activity, CheckCircle2, ChevronLeft, ChevronRight, Copy, Download, FolderOpen, MoreHorizontal, Plus, RefreshCcw, RotateCcw, Save, Search, Server, SlidersHorizontal, Trash2, X} from 'lucide-react';
import {Field, SecretField} from '../components/fields';
import {auxLabels} from '../constants';
import {PlatformsPage} from './PlatformsPage';
import {SoulPage} from './SoulPage';
import type {AppState, AuxModel, BatchProfileConfigRequest, BatchProfileConfigResult, BundledContentSyncRequest, BundledContentSyncResult, DingTalkSettings, EnvVar, ModelConfig, ModelOption, OperationsTab, PlatformKey, ProviderConfig, ProviderEntry, RuntimeProfileStatus, SkillDetail, SkillHubDetail, SkillHubQuery, SkillHubState, SkillsState, WizardStep} from '../types';
import {ensureCurrentModelOption, firstProviderID, modelOptionKey, nextProviderID, profileStatusText, providerIDs, providerReferenceLabels, slugProfileID} from '../utils';
import {assistantStatusClass, assistantStatusLabel, createProfileValidationMessage, formatBytes, skillSummaryLabel, suggestProfileID, wizardStepHelp} from './assistantUtils';
import {AssistantWizard} from './AssistantWizard';
import {AuxiliaryModelsPanel} from './AuxiliaryModelsPanel';
import {ProvidersPage} from './ProvidersPage';
import {SkillsPanel} from './SkillsPanel';
import {ProfileBatchTools} from './ProfileBatchTools';

export function AssistantsPage(props: {
    state: AppState;
    env: EnvVar[];
    setEnv: (value: EnvVar[]) => void;
    dingTalkSettings: DingTalkSettings;
    setDingTalkSettings: (value: DingTalkSettings) => void;
    providers: ProviderConfig;
    setProviders: (value: ProviderConfig) => void;
    selectedProvider: string;
    setSelectedProvider: (value: string) => void;
    model: ModelConfig | null;
    setModel: (value: ModelConfig) => void;
    modelOptions: ModelOption[];
    modelListStatus: string;
    modelTestStatus: string;
    selectedAux: string;
    setSelectedAux: (value: string) => void;
    auxModelOptions: Record<string, ModelOption[]>;
    auxModelListStatus: string;
    busy: boolean;
    showApiKey: boolean;
    setShowApiKey: (value: boolean) => void;
    newProfileID: string;
    setNewProfileID: (value: string) => void;
    newProfileName: string;
    setNewProfileName: (value: string) => void;
    newProfileCopyMode: string;
    setNewProfileCopyMode: (value: string) => void;
    newProfileEnabled: boolean;
    setNewProfileEnabled: (value: boolean) => void;
    wizardStep: WizardStep | null;
    setWizardStep: (value: WizardStep | null) => void;
    soulContent: string;
    setSoulContent: (value: string) => void;
    soulStatus: string;
    soulDirty: boolean;
    setSoulDirty: (value: boolean) => void;
    qrData: string;
    qrStatus: string;
	qrPlatform: PlatformKey | '';
    modelDirty: boolean;
    platformDirty: boolean;
    selectedPlatform: PlatformKey;
    setSelectedPlatform: (value: PlatformKey) => void;
    needsRebuild: boolean;
    hasPlatformBinding: boolean;
    skillsState: SkillsState | null;
    skillDetail: SkillDetail | null;
    skillsStatus: string;
    skillHubState: SkillHubState | null;
    skillHubDetail: SkillHubDetail | null;
    skillHubStatus: string;
    onSelect: (id: string) => Promise<boolean>;
    onCreate: () => Promise<boolean>;
    onRename: (id: string, name: string) => Promise<boolean>;
    onEnabled: (id: string, enabled: boolean) => void;
    onMove: (id: string, direction: string) => void;
    onDelete: (id: string) => Promise<boolean>;
    onSaveModelService: () => Promise<boolean>;
    onFetchModels: () => void;
    onFetchProviderModels: (providerID: string, provider: ProviderEntry) => void;
    onFetchAuxModels: (providerID: string) => void;
    onTestModel: () => void;
    onSaveSoul: () => Promise<boolean>;
    onDiscardSoul: () => void;
    onRestoreDefaultSoul: () => Promise<boolean>;
    onWeixinLogin: () => void;
    onCancelWeixin: () => void;
	onFeishuLogin: () => void;
	onCancelFeishu: () => void;
	onDingTalkLogin: () => void;
	onCancelDingTalk: () => void;
    onSaveWeCom: () => Promise<boolean>;
    onSaveFeishu: () => Promise<boolean>;
	onSaveDingTalk: () => Promise<boolean>;
    onApplyRecommendedDingTalkSettings: () => Promise<boolean>;
    onUnbindPlatform: (platform: PlatformKey) => void;
    onSaveCurrentPlatform: () => Promise<boolean>;
    onFinishSetup: (apply: boolean) => Promise<boolean>;
    onRebuild: () => void;
    onOpenOperations: (tab: OperationsTab) => void;
    onRefreshSkills: () => void;
    onSyncBundledSkills: () => Promise<boolean>;
    onRestoreDefaultSkills: () => Promise<boolean>;
    onSkillDetail: (path: string) => void;
    onDeleteSkill: (path: string) => Promise<boolean>;
    onBatchDeleteSkills: (paths: string[]) => Promise<boolean>;
    onOpenSkillDirectory: (path: string) => void;
    onSearchSkillHub: (query: SkillHubQuery) => void;
    onSkillHubDetail: (slug: string) => void;
    onInstallSkillHubSkill: (slug: string) => Promise<boolean>;
    onSkillsModeChange: (enabled: boolean) => void;
    onBatchCopyProfiles: (request: BatchProfileConfigRequest) => Promise<BatchProfileConfigResult | null>;
    onSyncBundledContent: (request: BundledContentSyncRequest) => Promise<BundledContentSyncResult | null>;
}) {
    const [showCreate, setShowCreate] = useState(false);
    const [editingID, setEditingID] = useState('');
    const [editingName, setEditingName] = useState('');
    const [deleteID, setDeleteID] = useState('');
    const [deleteConfirmText, setDeleteConfirmText] = useState('');
    const [showAdvancedModels, setShowAdvancedModels] = useState(false);
    const [showSkills, setShowSkills] = useState(false);
    const [showProviders, setShowProviders] = useState(false);
    const [managementTool, setManagementTool] = useState<'copy' | 'sync' | null>(null);
    const profiles = props.state.profiles?.profiles || [];
    const activeProfile = profiles.find((profile) => profile.id === props.state.activeProfile) || profiles[0];
    const activeIndex = activeProfile ? profiles.findIndex((profile) => profile.id === activeProfile.id) : -1;
    const activeStatus = activeProfile ? props.state.profileStatus?.profiles?.[activeProfile.id] : undefined;
    const activeSetupDone = !!activeProfile?.setupCompletedAt;
    const showAssistantManagement = activeSetupDone || profiles.length > 1;
    const activeWizardStep = activeSetupDone ? props.wizardStep : (props.wizardStep || 'model');
    const profileIDExists = profiles.some((profile) => profile.id === props.newProfileID);
    const profileIDValid = /^[a-z0-9](?:[a-z0-9-]{0,38}[a-z0-9])$/.test(props.newProfileID) && props.newProfileID !== 'default';
    const profileNameReady = props.newProfileName.trim() !== '';
    const canCreate = profileNameReady && profileIDValid && !profileIDExists;
    const createStarted = props.newProfileName.trim() !== '' || props.newProfileID.trim() !== '';
    const createValidationMessage = createStarted ? createProfileValidationMessage(props.newProfileName, props.newProfileID, profileIDExists) : '';

    useEffect(() => {
        props.onSkillsModeChange(showSkills);
        return () => props.onSkillsModeChange(false);
    }, [showSkills]);

    const startCreate = () => {
        setShowCreate(true);
        props.setNewProfileID('');
        props.setNewProfileName('');
        props.setNewProfileCopyMode('clean');
        props.setNewProfileEnabled(true);
    };

    const saveRename = async (id: string) => {
        if (!(await props.onRename(id, editingName))) return;
        setEditingID('');
        setEditingName('');
    };

    const startWizard = (step: WizardStep) => {
        setShowAdvancedModels(false);
        setShowSkills(false);
        setShowProviders(false);
        props.setWizardStep(step);
    };

    const openProviders = () => {
        setShowAdvancedModels(false);
        setShowSkills(false);
        setShowProviders(true);
        props.setWizardStep(null);
    };

    const selectAssistant = async (id: string) => {
        const target = profiles.find((profile) => profile.id === id);
        if (!(await props.onSelect(id))) return;
        setShowAdvancedModels(false);
        setShowSkills(false);
        setShowProviders(false);
        props.setWizardStep(target?.setupCompletedAt ? null : 'model');
    };

    return (
        <section className={`assistant-layout ${showSkills ? 'skills-mode' : ''}`}>
            {showAssistantManagement && activeProfile && (
                <div className="assistant-switcher">
                    <div>
                        <p className="eyebrow">助手管理</p>
                        <h2>切换和管理助手</h2>
                    </div>
                    <span className={`profile-status ${assistantStatusClass(activeProfile.setupCompletedAt, activeStatus, activeProfile.enabled, props.needsRebuild)}`}>{assistantStatusLabel(activeProfile.setupCompletedAt, activeStatus, activeProfile.enabled, props.needsRebuild)}</span>
                    <label className="assistant-select">
                        <span>切换助手</span>
                        <select value={activeProfile?.id || ''} onChange={(event) => selectAssistant(event.target.value)} disabled={props.busy}>
                            {profiles.map((profile) => <option key={profile.id} value={profile.id}>{profile.name || profile.id}</option>)}
                        </select>
                    </label>
                    <button className="ghost" onClick={startCreate} disabled={props.busy}><Plus size={16}/>新建助手</button>
                    <button className="ghost assistant-batch-action" onClick={() => setManagementTool('copy')} disabled={props.busy || profiles.length < 2}><Copy size={16}/>批量配置</button>
                    <button className="ghost assistant-batch-action" onClick={() => setManagementTool('sync')} disabled={props.busy}><RefreshCcw size={16}/>同步内置内容{props.state.bundledContent?.available ? <span className="action-dot"/> : null}</button>
                    <details className="more-menu">
                        <summary title="管理当前助手"><MoreHorizontal size={17}/></summary>
                        <div className="more-menu-popover">
                            <button onClick={() => {
                                setEditingID(activeProfile.id);
                                setEditingName(activeProfile.name || activeProfile.id);
                            }} disabled={props.busy}>重命名</button>
                            <button onClick={() => props.onEnabled(activeProfile.id, !activeProfile.enabled)} disabled={props.busy}>{activeProfile.enabled ? '停用助手' : '启用助手'}</button>
                            <button onClick={() => props.onMove(activeProfile.id, 'up')} disabled={props.busy || activeIndex <= 0}>上移</button>
                            <button onClick={() => props.onMove(activeProfile.id, 'down')} disabled={props.busy || activeIndex < 0 || activeIndex === profiles.length - 1}>下移</button>
                            <button className="danger-inline" onClick={() => {
                                setDeleteID(activeProfile.id);
                                setDeleteConfirmText('');
                            }} disabled={props.busy || activeProfile.id === 'default'}>删除</button>
                        </div>
                    </details>
                </div>
            )}
            {showAssistantManagement && activeProfile && editingID === activeProfile.id && (
                <div className="assistant-inline-editor">
                    <input value={editingName} onChange={(event) => setEditingName(event.target.value)} autoFocus disabled={props.busy}/>
                    <button className="primary inline-primary" onClick={() => saveRename(activeProfile.id)} disabled={props.busy || editingName.trim() === ''}>保存</button>
                    <button className="ghost" onClick={() => setEditingID('')} disabled={props.busy}>取消</button>
                </div>
            )}
            {showAssistantManagement && activeProfile && deleteID === activeProfile.id && (
                <div className="assistant-inline-editor danger-confirm">
                    <input value={deleteConfirmText} onChange={(event) => setDeleteConfirmText(event.target.value)} placeholder={`输入 ${activeProfile.id} 确认删除`} disabled={props.busy}/>
                    <button className="danger-button compact" onClick={async () => {
                        if (!(await props.onDelete(activeProfile.id))) return;
                        setDeleteID('');
                    }} disabled={props.busy || deleteConfirmText !== activeProfile.id}><Trash2 size={16}/>确认删除</button>
                    <button className="ghost" onClick={() => setDeleteID('')} disabled={props.busy}>取消</button>
                </div>
            )}
            {showAssistantManagement && showCreate && (
                <div className="assistant-create-overlay" role="presentation">
                    <aside className="assistant-create-drawer" role="dialog" aria-modal="true" aria-labelledby="create-assistant-title">
                        <div className="assistant-create-head">
                            <div>
                                <p className="eyebrow">新建助手</p>
                                <h2 id="create-assistant-title">创建一个新的助手</h2>
                            </div>
                            <button className="icon-button icon-only" type="button" onClick={() => setShowCreate(false)} disabled={props.busy} aria-label="关闭新建助手">
                                <X size={17}/>
                            </button>
                        </div>
                        <div className="assistant-create-form">
                            <Field label="显示名" value={props.newProfileName} onChange={(value) => {
                                props.setNewProfileName(value);
                                if (!props.newProfileID) props.setNewProfileID(suggestProfileID(profiles, value));
                            }}/>
                            <div>
                                <Field label="助手 ID" value={props.newProfileID} onChange={(value) => props.setNewProfileID(slugProfileID(value))}/>
                                <div className="field-hint">用于目录名和运行标识，创建后不可修改。</div>
                            </div>
                            <div className="create-choice-group" role="radiogroup" aria-label="创建方式">
                                <span>创建方式</span>
                                <label className={`create-choice ${props.newProfileCopyMode === 'clean' ? 'selected' : ''}`}>
                                    <input type="radio" name="profile-copy-mode" value="clean" checked={props.newProfileCopyMode === 'clean'} onChange={() => props.setNewProfileCopyMode('clean')}/>
                                    <strong>全新助手</strong>
                                    <em>不复制密钥、平台账号、记忆或会话。</em>
                                </label>
                                <label className={`create-choice ${props.newProfileCopyMode === 'personality-skills' ? 'selected' : ''}`}>
                                    <input type="radio" name="profile-copy-mode" value="personality-skills" checked={props.newProfileCopyMode === 'personality-skills'} onChange={() => props.setNewProfileCopyMode('personality-skills')}/>
                                    <strong>复制人格和技能</strong>
                                    <em>只复制 SOUL.md 和 skills，模型、密钥和平台绑定仍需单独配置。</em>
                                </label>
                            </div>
                            <label className="mini-toggle profile-enable"><input type="checkbox" checked={props.newProfileEnabled} onChange={(event) => props.setNewProfileEnabled(event.target.checked)}/>创建后加入运行列表</label>
                            <div className="field-hint">未绑定平台时不会启动消息入口。</div>
                            {createValidationMessage && <div className="form-warning">{createValidationMessage}</div>}
                        </div>
                        <div className="assistant-create-actions">
                            <button className="ghost" type="button" onClick={() => setShowCreate(false)} disabled={props.busy}>取消</button>
                            <button className="primary no-margin" type="button" onClick={async () => {
                                if (!(await props.onCreate())) return;
                                setShowCreate(false);
                                setShowAdvancedModels(false);
                                setShowSkills(false);
                                setShowProviders(false);
                                props.setWizardStep('model');
                            }} disabled={props.busy || !canCreate}><Plus size={16}/>创建助手</button>
                        </div>
                    </aside>
                </div>
            )}
            {showAssistantManagement && managementTool && (
                <ProfileBatchTools
                    mode={managementTool}
                    profiles={profiles}
                    activeProfile={activeProfile?.id || 'default'}
                    busy={props.busy}
                    onClose={() => setManagementTool(null)}
                    onCopy={props.onBatchCopyProfiles}
                    onSync={props.onSyncBundledContent}
                />
            )}

            <div className="assistant-detail">
                {!activeProfile && <div className="panel">暂无助手。</div>}
                {activeProfile && !activeWizardStep && !showAdvancedModels && !showSkills && !showProviders && (
                    <AssistantSummary
                        profileName={activeProfile.name || activeProfile.id}
                        setupCompletedAt={activeProfile.setupCompletedAt || ''}
                        enabled={activeProfile.enabled}
                        status={activeStatus}
                        model={props.model}
                        providers={props.providers}
                        hasPlatformBinding={props.hasPlatformBinding}
                        skillsState={props.skillsState}
                        needsRebuild={props.needsRebuild}
                        busy={props.busy}
                        onStep={startWizard}
                        onAdvancedModels={() => setShowAdvancedModels(true)}
                        onProviders={openProviders}
                        onSkills={() => {
                            setShowCreate(false);
                            setEditingID('');
                            setDeleteID('');
                            setShowAdvancedModels(false);
                            setShowProviders(false);
                            setShowSkills(true);
                            props.onRefreshSkills();
                        }}
                        onEnabled={(enabled) => props.onEnabled(activeProfile.id, enabled)}
                        onRebuild={props.onRebuild}
                        onOpenOperations={props.onOpenOperations}
                    />
                )}
                {activeProfile && !activeWizardStep && showSkills && (
                    <SkillsPanel
                        profileName={activeProfile.name || activeProfile.id}
                        skillsState={props.skillsState}
                        detail={props.skillDetail}
                        status={props.skillsStatus}
                        hubState={props.skillHubState}
                        hubDetail={props.skillHubDetail}
                        hubStatus={props.skillHubStatus}
                        busy={props.busy}
                        onBack={() => setShowSkills(false)}
                        onRefresh={props.onRefreshSkills}
                        onSyncBundledSkills={props.onSyncBundledSkills}
                        onRestoreDefaultSkills={props.onRestoreDefaultSkills}
                        onDetail={props.onSkillDetail}
                        onDelete={props.onDeleteSkill}
                        onDeleteMany={props.onBatchDeleteSkills}
                        onOpenDirectory={props.onOpenSkillDirectory}
                        onSearchHub={props.onSearchSkillHub}
                        onHubDetail={props.onSkillHubDetail}
                        onInstallHubSkill={props.onInstallSkillHubSkill}
                    />
                )}
                {activeProfile && !activeWizardStep && showAdvancedModels && props.model && (
                    <AuxiliaryModelsPanel
                        model={props.model}
                        setModel={props.setModel}
                        providers={props.providers}
                        selectedAux={props.selectedAux}
                        setSelectedAux={props.setSelectedAux}
                        modelOptions={props.modelOptions}
                        auxModelOptions={props.auxModelOptions}
                        auxModelListStatus={props.auxModelListStatus}
                        busy={props.busy}
                        onFetchAuxModels={props.onFetchAuxModels}
                        onSave={props.onSaveModelService}
                        onBack={() => setShowAdvancedModels(false)}
                    />
                )}
                {activeProfile && !activeWizardStep && showProviders && (
                    <ProvidersPage
                        providers={props.providers}
                        setProviders={props.setProviders}
                        selectedProvider={props.selectedProvider}
                        setSelectedProvider={props.setSelectedProvider}
                        model={props.model}
                        busy={props.busy}
                        showApiKey={props.showApiKey}
                        setShowApiKey={props.setShowApiKey}
                        modelOptions={props.modelOptions}
                        modelListStatus={props.modelListStatus}
                        onFetchModels={(provider) => {
                            const id = Object.entries(props.providers.providers).find(([, p]) => p === provider)?.[0] || props.selectedProvider;
                            props.onFetchProviderModels(id, provider);
                        }}
                        onSave={props.onSaveModelService}
                        onBack={() => setShowProviders(false)}
                    />
                )}
                {activeProfile && activeWizardStep && (
                    <AssistantWizard
                        step={activeWizardStep}
                        setupDone={activeSetupDone}
                        profileID={activeProfile.id}
                        profileName={activeProfile.name || activeProfile.id}
                        env={props.env}
                        dingTalkSettings={props.dingTalkSettings}
                        setDingTalkSettings={props.setDingTalkSettings}
                        providers={props.providers}
                        setProviders={props.setProviders}
                        selectedProvider={props.selectedProvider}
                        setSelectedProvider={props.setSelectedProvider}
                        model={props.model}
                        setModel={props.setModel}
                        modelOptions={props.modelOptions}
                        modelListStatus={props.modelListStatus}
                        modelTestStatus={props.modelTestStatus}
                        busy={props.busy}
                        showApiKey={props.showApiKey}
                        setShowApiKey={props.setShowApiKey}
                        soulContent={props.soulContent}
                        setSoulContent={props.setSoulContent}
                        soulStatus={props.soulStatus}
                        soulDirty={props.soulDirty}
                        setSoulDirty={props.setSoulDirty}
                        qrData={props.qrData}
                        qrStatus={props.qrStatus}
						qrPlatform={props.qrPlatform}
                        modelDirty={props.modelDirty}
                        platformDirty={props.platformDirty}
                        selectedPlatform={props.selectedPlatform}
                        setSelectedPlatform={props.setSelectedPlatform}
                        setEnv={props.setEnv}
                        hasPlatformBinding={props.hasPlatformBinding}
                        onStep={props.setWizardStep}
                        onSaveModelService={props.onSaveModelService}
                        onFetchModels={props.onFetchModels}
                        onTestModel={props.onTestModel}
                        onSaveSoul={props.onSaveSoul}
                        onDiscardSoul={props.onDiscardSoul}
                        onRestoreDefaultSoul={props.onRestoreDefaultSoul}
                        onWeixinLogin={props.onWeixinLogin}
                        onCancelWeixin={props.onCancelWeixin}
						onFeishuLogin={props.onFeishuLogin}
						onCancelFeishu={props.onCancelFeishu}
						onDingTalkLogin={props.onDingTalkLogin}
						onCancelDingTalk={props.onCancelDingTalk}
                        onSaveWeCom={props.onSaveWeCom}
                        onSaveFeishu={props.onSaveFeishu}
						onSaveDingTalk={props.onSaveDingTalk}
                        onApplyRecommendedDingTalkSettings={props.onApplyRecommendedDingTalkSettings}
                        onUnbindPlatform={props.onUnbindPlatform}
                        onSaveCurrentPlatform={props.onSaveCurrentPlatform}
                        onFinishSetup={props.onFinishSetup}
                        onOpenProviders={openProviders}
                    />
                )}
            </div>
        </section>
    );
}

function AssistantSummary(props: {
    profileName: string;
    setupCompletedAt: string;
    enabled: boolean;
    status?: RuntimeProfileStatus;
    model: ModelConfig | null;
    providers: ProviderConfig;
    hasPlatformBinding: boolean;
    skillsState: SkillsState | null;
    needsRebuild: boolean;
    busy: boolean;
    onStep: (step: WizardStep) => void;
    onAdvancedModels: () => void;
    onSkills: () => void;
    onProviders: () => void;
    onEnabled: (enabled: boolean) => void;
    onRebuild: () => void;
    onOpenOperations: (tab: OperationsTab) => void;
}) {
    const provider = props.model ? props.providers.providers[props.model.provider] : undefined;
    const skills = props.skillsState;
    return (
        <div className="assistant-summary">
            <div className="setup-card">
                <div>
                    <p className="eyebrow">当前助手</p>
                    <h2>{props.profileName}</h2>
                    <p className="setup-subtitle">需要修改时，直接进入对应配置项；保存后按提示应用即可。</p>
                </div>
                <div className="setup-status-list">
                    <button onClick={() => props.onStep('model')}>
                        <span>模型服务</span>
                        <strong>{provider?.label || '未选择'} · {props.model?.default || '未选择模型'}</strong>
                    </button>
                    <button onClick={() => props.onStep('soul')}>
                        <span>人格设定</span>
                        <strong>{props.setupCompletedAt ? '已完成' : '未完成'}</strong>
                    </button>
                    <button onClick={() => props.onStep('platforms')}>
                        <span>平台绑定</span>
                        <strong>{props.hasPlatformBinding ? '已绑定' : '暂未绑定'}</strong>
                    </button>
                    <button onClick={props.onSkills}>
                        <span>技能</span>
                        <strong>{skills ? skillSummaryLabel(skills) : '正在读取'}</strong>
                    </button>
                </div>
                <div className="setup-actions">
                    <label className="mini-toggle"><input type="checkbox" checked={props.enabled} onChange={(event) => props.onEnabled(event.target.checked)} disabled={props.busy}/>启用助手</label>
                    <button className="ghost" onClick={props.onAdvancedModels} disabled={props.busy}><SlidersHorizontal size={16}/>高级模型设置</button>
                    <button className="ghost" onClick={props.onProviders} disabled={props.busy}><Server size={16}/>供应商管理</button>
                    <button className="ghost" onClick={() => props.onOpenOperations('runtime')} disabled={props.busy}>运行状态：{profileStatusText(props.status?.state, props.enabled)}</button>
                    <button className="primary no-margin" onClick={() => props.onStep('model')} disabled={props.busy}>重新配置</button>
                    {props.needsRebuild && <button className="primary no-margin" onClick={props.onRebuild} disabled={props.busy}><RefreshCcw size={16}/>应用配置</button>}
                </div>
            </div>
        </div>
    );
}
