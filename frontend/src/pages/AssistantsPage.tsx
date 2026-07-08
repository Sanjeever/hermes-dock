import {useEffect, useState} from 'react';
import {Activity, CheckCircle2, ChevronLeft, ChevronRight, Download, FolderOpen, MoreHorizontal, Plus, RefreshCcw, Save, Search, SlidersHorizontal, Trash2} from 'lucide-react';
import {Field, SecretField} from '../components/fields';
import {auxLabels} from '../constants';
import {PlatformsPage} from './PlatformsPage';
import {SoulPage} from './SoulPage';
import type {AppState, AuxModel, EnvVar, ModelConfig, ModelOption, OperationsTab, PlatformKey, ProviderConfig, ProviderEntry, RuntimeProfileStatus, SkillDetail, SkillHubDetail, SkillHubQuery, SkillHubState, SkillsState, WizardStep} from '../types';
import {ensureCurrentModelOption, firstProviderID, modelOptionKey, nextProviderID, profileStatusText, providerIDs, providerReferenceLabels, slugProfileID, statusClassName} from '../utils';

export function AssistantsPage(props: {
    state: AppState;
    env: EnvVar[];
    setEnv: (value: EnvVar[]) => void;
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
    onFetchAuxModels: (providerID: string) => void;
    onTestModel: () => void;
    onSaveSoul: () => Promise<boolean>;
    onDiscardSoul: () => void;
    onWeixinLogin: () => void;
    onCancelWeixin: () => void;
    onSaveWeCom: () => Promise<boolean>;
    onSaveFeishu: () => Promise<boolean>;
    onUnbindPlatform: (platform: PlatformKey) => void;
    onSaveCurrentPlatform: () => Promise<boolean>;
    onFinishSetup: (apply: boolean) => Promise<boolean>;
    onRebuild: () => void;
    onOpenOperations: (tab: OperationsTab) => void;
    onRefreshSkills: () => void;
    onSyncBundledSkills: () => Promise<boolean>;
    onSkillDetail: (path: string) => void;
    onDeleteSkill: (path: string) => Promise<boolean>;
    onOpenSkillDirectory: (path: string) => void;
    onSearchSkillHub: (query: SkillHubQuery) => void;
    onSkillHubDetail: (slug: string) => void;
    onInstallSkillHubSkill: (slug: string) => Promise<boolean>;
    onSkillsModeChange: (enabled: boolean) => void;
}) {
    const [showCreate, setShowCreate] = useState(false);
    const [editingID, setEditingID] = useState('');
    const [editingName, setEditingName] = useState('');
    const [deleteID, setDeleteID] = useState('');
    const [deleteConfirmText, setDeleteConfirmText] = useState('');
    const [showAdvancedModels, setShowAdvancedModels] = useState(false);
    const [showSkills, setShowSkills] = useState(false);
    const profiles = props.state.profiles?.profiles || [];
    const activeProfile = profiles.find((profile) => profile.id === props.state.activeProfile) || profiles[0];
    const activeIndex = activeProfile ? profiles.findIndex((profile) => profile.id === activeProfile.id) : -1;
    const activeStatus = activeProfile ? props.state.profileStatus?.profiles?.[activeProfile.id] : undefined;
    const activeSetupDone = !!activeProfile?.setupCompletedAt;
    const activeWizardStep = activeSetupDone ? props.wizardStep : (props.wizardStep || 'model');
    const profileIDExists = profiles.some((profile) => profile.id === props.newProfileID);
    const profileIDValid = /^[a-z0-9](?:[a-z0-9-]{0,38}[a-z0-9])$/.test(props.newProfileID) && props.newProfileID !== 'default';
    const canCreate = profileIDValid && !profileIDExists;

    useEffect(() => {
        props.onSkillsModeChange(showSkills);
        return () => props.onSkillsModeChange(false);
    }, [showSkills]);

    const startCreate = () => {
        setShowCreate(true);
        props.setNewProfileID('');
        props.setNewProfileName('');
    };

    const saveRename = async (id: string) => {
        if (!(await props.onRename(id, editingName))) return;
        setEditingID('');
        setEditingName('');
    };

    const startWizard = (step: WizardStep) => {
        setShowAdvancedModels(false);
        setShowSkills(false);
        props.setWizardStep(step);
    };

    const selectAssistant = async (id: string) => {
        const target = profiles.find((profile) => profile.id === id);
        if (!(await props.onSelect(id))) return;
        setShowAdvancedModels(false);
        setShowSkills(false);
        props.setWizardStep(target?.setupCompletedAt ? null : 'model');
    };

    return (
        <section className={`assistant-layout ${showSkills ? 'skills-mode' : ''}`}>
            {activeSetupDone && (
                <div className="assistant-switcher">
                    {activeProfile && (
                        <div>
                            <p className="eyebrow">当前助手</p>
                            <h2>{activeProfile.name || activeProfile.id}</h2>
                        </div>
                    )}
                    {activeProfile && <span className={`profile-status ${assistantStatusClass(activeProfile.setupCompletedAt, activeStatus, activeProfile.enabled, props.needsRebuild)}`}>{assistantStatusLabel(activeProfile.setupCompletedAt, activeStatus, activeProfile.enabled, props.needsRebuild)}</span>}
                    <label className="assistant-select">
                        <span>切换助手</span>
                        <select value={activeProfile?.id || ''} onChange={(event) => selectAssistant(event.target.value)} disabled={props.busy}>
                            {profiles.map((profile) => <option key={profile.id} value={profile.id}>{profile.name || profile.id}</option>)}
                        </select>
                    </label>
                    <button className="ghost" onClick={startCreate} disabled={props.busy}><Plus size={16}/>新建助手</button>
                    {activeProfile && (
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
                    )}
                </div>
            )}
            {activeSetupDone && activeProfile && editingID === activeProfile.id && (
                <div className="assistant-inline-editor">
                    <input value={editingName} onChange={(event) => setEditingName(event.target.value)} autoFocus disabled={props.busy}/>
                    <button className="primary inline-primary" onClick={() => saveRename(activeProfile.id)} disabled={props.busy || editingName.trim() === ''}>保存</button>
                    <button className="ghost" onClick={() => setEditingID('')} disabled={props.busy}>取消</button>
                </div>
            )}
            {activeSetupDone && activeProfile && deleteID === activeProfile.id && (
                <div className="assistant-inline-editor danger-confirm">
                    <input value={deleteConfirmText} onChange={(event) => setDeleteConfirmText(event.target.value)} placeholder={`输入 ${activeProfile.id} 确认删除`} disabled={props.busy}/>
                    <button className="danger-button compact" onClick={async () => {
                        if (!(await props.onDelete(activeProfile.id))) return;
                        setDeleteID('');
                    }} disabled={props.busy || deleteConfirmText !== activeProfile.id}><Trash2 size={16}/>确认删除</button>
                    <button className="ghost" onClick={() => setDeleteID('')} disabled={props.busy}>取消</button>
                </div>
            )}
            {activeSetupDone && showCreate && (
                <div className="create-drawer">
                    <p className="eyebrow">新建助手</p>
                    <Field label="显示名" value={props.newProfileName} onChange={(value) => {
                        props.setNewProfileName(value);
                        if (!props.newProfileID) props.setNewProfileID(suggestProfileID(profiles, value));
                    }}/>
                    <Field label="助手 ID" value={props.newProfileID} onChange={(value) => props.setNewProfileID(slugProfileID(value))}/>
                    <label className="field">
                        <span>创建方式</span>
                        <select value={props.newProfileCopyMode} onChange={(event) => props.setNewProfileCopyMode(event.target.value)}>
                            <option value="clean">全新配置</option>
                            <option value="personality-skills">复制当前助手的人格和技能</option>
                        </select>
                    </label>
                    <label className="mini-toggle profile-enable"><input type="checkbox" checked={props.newProfileEnabled} onChange={(event) => props.setNewProfileEnabled(event.target.checked)}/>创建后启用助手</label>
                    {!canCreate && (props.newProfileID || props.newProfileName) && (
                        <div className="form-warning">{profileIDExists ? '该助手 ID 已存在，请换一个。' : '助手 ID 只能包含小写字母、数字和连字符，且不能使用 default。'}</div>
                    )}
                    <div className="actions">
                        <button className="primary no-margin" onClick={async () => {
                            if (!(await props.onCreate())) return;
                            setShowCreate(false);
                            setShowAdvancedModels(false);
                            setShowSkills(false);
                            props.setWizardStep('model');
                        }} disabled={props.busy || !canCreate}><Save size={16}/>开始配置</button>
                        <button className="ghost" onClick={() => setShowCreate(false)} disabled={props.busy}>取消</button>
                    </div>
                </div>
            )}

            <div className="assistant-detail">
                {!activeProfile && <div className="panel">暂无助手。</div>}
                {activeProfile && !activeWizardStep && !showAdvancedModels && !showSkills && (
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
                        onSkills={() => {
                            setShowCreate(false);
                            setEditingID('');
                            setDeleteID('');
                            setShowAdvancedModels(false);
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
                        onDetail={props.onSkillDetail}
                        onDelete={props.onDeleteSkill}
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
                {activeProfile && activeWizardStep && (
                    <AssistantWizard
                        step={activeWizardStep}
                        setupDone={activeSetupDone}
                        profileID={activeProfile.id}
                        profileName={activeProfile.name || activeProfile.id}
                        env={props.env}
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
                        onWeixinLogin={props.onWeixinLogin}
                        onCancelWeixin={props.onCancelWeixin}
                        onSaveWeCom={props.onSaveWeCom}
                        onSaveFeishu={props.onSaveFeishu}
                        onUnbindPlatform={props.onUnbindPlatform}
                        onSaveCurrentPlatform={props.onSaveCurrentPlatform}
                        onFinishSetup={props.onFinishSetup}
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
                    <p className="eyebrow">助手已就绪</p>
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
                    <button className="ghost" onClick={() => props.onOpenOperations('runtime')} disabled={props.busy}>运行状态：{profileStatusText(props.status?.state, props.enabled)}</button>
                    <button className="primary no-margin" onClick={() => props.onStep('model')} disabled={props.busy}>重新配置</button>
                    {props.needsRebuild && <button className="primary no-margin" onClick={props.onRebuild} disabled={props.busy}><RefreshCcw size={16}/>应用并重建</button>}
                </div>
            </div>
        </div>
    );
}

function SkillsPanel(props: {
    profileName: string;
    skillsState: SkillsState | null;
    detail: SkillDetail | null;
    status: string;
    hubState: SkillHubState | null;
    hubDetail: SkillHubDetail | null;
    hubStatus: string;
    busy: boolean;
    onBack: () => void;
    onRefresh: () => void;
    onSyncBundledSkills: () => Promise<boolean>;
    onDetail: (path: string) => void;
    onDelete: (path: string) => Promise<boolean>;
    onOpenDirectory: (path: string) => void;
    onSearchHub: (query: SkillHubQuery) => void;
    onHubDetail: (slug: string) => void;
    onInstallHubSkill: (slug: string) => Promise<boolean>;
}) {
    const [view, setView] = useState<'local' | 'hub'>('local');
    const [query, setQuery] = useState('');
    const [filter, setFilter] = useState<'all' | 'builtin' | 'custom' | 'conflict'>('all');
    const [detailTab, setDetailTab] = useState<'overview' | 'skill' | 'files'>('overview');
    const [deletePath, setDeletePath] = useState('');
    const [hubKeyword, setHubKeyword] = useState('');
    const [hubCategory, setHubCategory] = useState('');
    const skills = props.skillsState?.skills || [];
    const normalizedQuery = query.trim().toLowerCase();
    const filtered = skills.filter((skill) => {
        if (filter === 'builtin' && !skill.builtin) return false;
        if (filter === 'custom' && skill.builtin) return false;
        if (filter === 'conflict' && !skill.conflict) return false;
        if (!normalizedQuery) return true;
        return [skill.name, skill.description, skill.category, skill.path].some((value) => (value || '').toLowerCase().includes(normalizedQuery));
    });
    const filteredPaths = filtered.map((skill) => skill.path).join('\n');
    const activeDetail = props.detail && filtered.some((skill) => skill.path === props.detail?.path) ? props.detail : null;
    const hubSkills = props.hubState?.skills || [];
    const hubSlugs = hubSkills.map((skill) => skill.slug).join('\n');
    const activeHubDetail = props.hubDetail && hubSkills.some((skill) => skill.slug === props.hubDetail?.slug) ? props.hubDetail : null;
    const makeHubQuery = (keyword = hubKeyword, category = hubCategory, page = 1): SkillHubQuery => ({
        keyword,
        category,
        page,
        pageSize: 24,
        sortBy: 'score',
        order: 'desc',
    });

    useEffect(() => {
        if (filtered.length === 0) return;
        if (props.detail && filtered.some((skill) => skill.path === props.detail?.path)) return;
        setDeletePath('');
        setDetailTab('overview');
        props.onDetail(filtered[0].path);
    }, [filteredPaths, props.detail?.path]);

    useEffect(() => {
        if (view !== 'hub' || props.hubState) return;
        props.onSearchHub(makeHubQuery());
    }, [view]);

    useEffect(() => {
        if (view !== 'hub' || hubSkills.length === 0) return;
        if (props.hubDetail && hubSkills.some((skill) => skill.slug === props.hubDetail?.slug)) return;
        props.onHubDetail(hubSkills[0].slug);
    }, [view, hubSlugs, props.hubDetail?.slug]);

    const searchHub = (keyword = hubKeyword, category = hubCategory, page = 1) => {
        props.onSearchHub(makeHubQuery(keyword, category, page));
    };
    const hubPage = props.hubState?.page || 1;
    const hubTotalPages = Math.max(1, Math.ceil((props.hubState?.total || 0) / (props.hubState?.pageSize || 24)));

    return (
        <div className="skills-panel">
            <div className="skills-shell">
                <div className="skills-head">
                    <div>
                        <p className="eyebrow">当前助手：{props.profileName}</p>
                        <h2>技能管理</h2>
                        <p>{view === 'local' ? skillSummaryLine(props.skillsState) : skillHubSummaryLine(props.hubState)}</p>
                    </div>
                    <div className="skills-view-toggle">
                        <button className={view === 'local' ? 'selected' : ''} onClick={() => setView('local')} disabled={props.busy}>本地</button>
                        <button className={view === 'hub' ? 'selected' : ''} onClick={() => setView('hub')} disabled={props.busy}>技能中心</button>
                    </div>
                    <div className="skills-head-actions">
                        {view === 'local' && <button className="ghost" onClick={props.onSyncBundledSkills} disabled={props.busy}><Download size={16}/>同步内置技能</button>}
                        <button className="ghost" onClick={() => view === 'local' ? props.onRefresh() : searchHub(hubKeyword, hubCategory, hubPage)} disabled={props.busy}><RefreshCcw size={16}/>刷新</button>
                        <button className="ghost" onClick={props.onBack} disabled={props.busy}><ChevronLeft size={16}/>返回摘要</button>
                    </div>
                </div>
                <div className="skills-controls">
                    {view === 'local' ? (
                        <div className="skills-toolbar">
                            <label className="skills-search">
                                <Search size={16}/>
                                <input value={query} onChange={(event) => setQuery(event.target.value)} placeholder="搜索技能名、描述或分类"/>
                            </label>
                            <div className="segmented compact">
                                {[
                                    ['all', '全部'],
                                    ['builtin', '内置'],
                                    ['custom', '自定义'],
                                    ['conflict', '冲突'],
                                ].map(([value, label]) => (
                                    <button key={value} className={filter === value ? 'selected' : ''} onClick={() => setFilter(value as typeof filter)}>{label}</button>
                                ))}
                            </div>
                        </div>
                    ) : (
                        <div className="skills-toolbar hub-toolbar">
                            <label className="skills-search">
                                <Search size={16}/>
                                <input value={hubKeyword} onChange={(event) => setHubKeyword(event.target.value)} onKeyDown={(event) => {
                                    if (event.key === 'Enter') searchHub();
                                }} placeholder="搜索技能中心"/>
                            </label>
                            <select value={hubCategory} onChange={(event) => {
                                const next = event.target.value;
                                setHubCategory(next);
                                searchHub(hubKeyword, next);
                            }} disabled={props.busy}>
                                <option value="">全部分类</option>
                                {(props.hubState?.categories || []).map((category) => (
                                    <option key={category.key} value={category.key}>{category.name}</option>
                                ))}
                            </select>
                            <button className="ghost" onClick={() => searchHub()} disabled={props.busy}>搜索</button>
                            {props.hubState && hubTotalPages > 1 && (
                                <div className="hub-pager compact">
                                    <button className="ghost" onClick={() => searchHub(hubKeyword, hubCategory, hubPage - 1)} disabled={props.busy || hubPage <= 1}>上一页</button>
                                    <span>{hubPage} / {hubTotalPages}</span>
                                    <button className="ghost" onClick={() => searchHub(hubKeyword, hubCategory, hubPage + 1)} disabled={props.busy || hubPage >= hubTotalPages}>下一页</button>
                                </div>
                            )}
                        </div>
                    )}
                    {view === 'local' && props.status && <div className="inline-status skills-status">{props.status}</div>}
                    {view === 'hub' && props.hubStatus && <div className="inline-status skills-status">{props.hubStatus}</div>}
                </div>
                {view === 'local' ? (
                    <div className="skills-workspace">
                        <aside className="skill-list-pane">
                            {filtered.length === 0 ? (
                                <div className="skill-list-empty">{skills.length === 0 ? '当前助手还没有技能。' : '没有匹配的技能。'}</div>
                            ) : (
                                <div className="skill-list">
                                    {filtered.map((skill) => (
                                        <button key={skill.path} className={`skill-row ${skill.conflict ? 'conflict' : ''} ${activeDetail?.path === skill.path ? 'selected' : ''}`} onClick={() => {
                                            setDeletePath('');
                                            setDetailTab('overview');
                                            props.onDetail(skill.path);
                                        }}>
                                            <span className="skill-row-main">
                                                <span className="skill-row-title">
                                                    <strong>{skill.name}</strong>
                                                    <Badge label={skill.builtin ? '内置' : '自定义'} tone={skill.builtin ? 'muted' : 'ok'}/>
                                                </span>
                                                <small>{skill.description || skill.error || '无描述'}</small>
                                            </span>
                                        </button>
                                    ))}
                                </div>
                            )}
                        </aside>
                        <section className="skill-detail-pane">
                            {!activeDetail ? (
                                <div className="skill-detail-empty">
                                    <p className="eyebrow">技能详情</p>
                                    <h2>{filtered.length === 0 ? '暂无技能' : '正在读取'}</h2>
                                    <p>{filtered.length === 0 ? '调整搜索或筛选条件后再查看。' : '详情会显示在这里。'}</p>
                                </div>
                            ) : (
                                <>
                                    <div className="skill-detail-head">
                                        <div>
                                            <p className="eyebrow">{activeDetail.builtin ? '内置技能' : '自定义技能'}</p>
                                            <h2>{activeDetail.name}</h2>
                                            <p>{activeDetail.description || activeDetail.error || '无描述'}</p>
                                        </div>
                                        <button className="ghost" onClick={() => props.onOpenDirectory(activeDetail.path)} disabled={props.busy}><FolderOpen size={16}/>打开目录</button>
                                    </div>
                                    <div className="skill-detail-tabs">
                                        {[
                                            ['overview', '概览'],
                                            ['skill', 'SKILL.md'],
                                            ['files', `文件 ${activeDetail.fileCount}`],
                                        ].map(([value, label]) => (
                                            <button key={value} className={detailTab === value ? 'selected' : ''} onClick={() => setDetailTab(value as typeof detailTab)}>{label}</button>
                                        ))}
                                    </div>
                                    {detailTab === 'overview' && (
                                        <div className="skill-detail-section">
                                            <dl className="skill-meta-list">
                                                <Meta label="路径" value={activeDetail.path}/>
                                                <Meta label="版本" value={activeDetail.version || '未声明'}/>
                                                <Meta label="作者" value={activeDetail.author || '未声明'}/>
                                                <Meta label="平台" value={activeDetail.platforms?.length ? activeDetail.platforms.join(', ') : '未限制'}/>
                                                <Meta label="标签" value={activeDetail.tags?.length ? activeDetail.tags.join(', ') : '未声明'}/>
                                                <Meta label="大小" value={formatBytes(activeDetail.sizeBytes)}/>
                                            </dl>
                                            {(activeDetail.conflictPaths?.length || 0) > 0 && (
                                                <div className="form-warning">发现同名技能：{activeDetail.conflictPaths.join('、')}</div>
                                            )}
                                            <div className="skill-danger-zone">
                                                <div>
                                                    <strong>危险操作</strong>
                                                    <p>删除前会自动备份，重建后生效。</p>
                                                </div>
                                                {deletePath === activeDetail.path ? (
                                                    <div className="skill-delete-confirm">
                                                        <span>确认删除 {activeDetail.name}？</span>
                                                        <button className="danger-button compact" onClick={async () => {
                                                            if (await props.onDelete(activeDetail.path)) setDeletePath('');
                                                        }} disabled={props.busy}><Trash2 size={16}/>确认删除</button>
                                                        <button className="ghost" onClick={() => setDeletePath('')} disabled={props.busy}>取消</button>
                                                    </div>
                                                ) : (
                                                    <button className="ghost danger-inline" onClick={() => setDeletePath(activeDetail.path)} disabled={props.busy}><Trash2 size={16}/>删除技能</button>
                                                )}
                                            </div>
                                        </div>
                                    )}
                                    {detailTab === 'skill' && (
                                        <pre className="skill-preview">{activeDetail.preview}{activeDetail.previewTruncated ? '\n\n...预览已截断' : ''}</pre>
                                    )}
                                    {detailTab === 'files' && (
                                        <div className="skill-files">
                                            {activeDetail.files.map((file) => (
                                                <div key={file.path}>
                                                    <code>{file.path}</code>
                                                    <span>{formatBytes(file.sizeBytes)}</span>
                                                </div>
                                            ))}
                                            {activeDetail.filesTruncated && <div className="muted">还有更多文件未显示。</div>}
                                        </div>
                                    )}
                                </>
                            )}
                        </section>
                    </div>
                ) : (
                    <div className="skills-workspace">
                        <aside className="skill-list-pane">
                            {hubSkills.length === 0 ? (
                                <div className="skill-list-empty">{props.hubStatus ? '正在读取技能中心。' : '没有匹配的技能。'}</div>
                            ) : (
                                <div className="skill-list">
                                    {hubSkills.map((skill) => (
                                        <button key={skill.slug} className={`skill-row ${activeHubDetail?.slug === skill.slug ? 'selected' : ''}`} onClick={() => {
                                            props.onHubDetail(skill.slug);
                                        }}>
                                            <span className="skill-row-main">
                                                <span className="skill-row-title">
                                                    <strong>{skill.name}</strong>
                                                    <Badge label={skill.installed ? '已安装' : skill.source || '技能中心'} tone={skill.installed ? 'ok' : 'muted'}/>
                                                </span>
                                                <small>{skill.description || '无描述'}</small>
                                            </span>
                                        </button>
                                    ))}
                                </div>
                            )}
                        </aside>
                        <section className="skill-detail-pane">
                            {!activeHubDetail ? (
                                <div className="skill-detail-empty">
                                    <p className="eyebrow">技能中心</p>
                                    <h2>{hubSkills.length === 0 ? '暂无结果' : '正在读取'}</h2>
                                    <p>搜索技能中心并安装到当前助手。</p>
                                </div>
                            ) : (
                                <>
                                    <div className="skill-detail-head">
                                        <div>
                                            <p className="eyebrow">{activeHubDetail.categoryName || '技能中心'}</p>
                                            <h2>{activeHubDetail.name}</h2>
                                            <p>{activeHubDetail.description || '无描述'}</p>
                                        </div>
                                        <button className={activeHubDetail.installed ? 'ghost' : 'primary no-margin'} onClick={async () => {
                                            if (activeHubDetail.installed) return;
                                            if (await props.onInstallHubSkill(activeHubDetail.slug)) {
                                                searchHub(hubKeyword, hubCategory, hubPage);
                                            }
                                        }} disabled={props.busy || activeHubDetail.installed}>
                                            <Download size={16}/>{activeHubDetail.installed ? '已安装' : '安装'}
                                        </button>
                                    </div>
                                    <dl className="skill-meta-list">
                                        <Meta label="版本" value={activeHubDetail.version || '未声明'}/>
                                        <Meta label="来源" value={activeHubDetail.source || '技能中心'}/>
                                        <Meta label="作者" value={activeHubDetail.ownerName || '未声明'}/>
                                        <Meta label="统计" value={`${formatCount(activeHubDetail.downloads)} 下载 · ${formatCount(activeHubDetail.stars)} 收藏`}/>
                                        <Meta label="密钥" value={activeHubDetail.requiresApiKey ? '需要 API Key' : '不需要 API Key'}/>
                                        <Meta label="文件" value={`${activeHubDetail.fileCount || activeHubDetail.files?.length || 0} 个`}/>
                                    </dl>
                                    {activeHubDetail.securityReports?.some((report) => report.status && report.status !== 'clean' && report.status !== 'safe') && (
                                        <div className="form-warning">安全报告提示存在潜在风险，请确认来源可信后再安装。</div>
                                    )}
                                    {activeHubDetail.installed && activeHubDetail.installedPath && (
                                        <button className="ghost skill-inline-action" onClick={() => props.onOpenDirectory(activeHubDetail.installedPath)} disabled={props.busy}><FolderOpen size={16}/>打开本地目录</button>
                                    )}
                                </>
                            )}
                        </section>
                    </div>
                )}
            </div>
        </div>
    );
}

function skillSummaryLine(state: SkillsState | null) {
    if (!state) return '正在读取技能';
    const parts = [`${state.total} 个技能`, `${state.builtinCount} 内置`, `${state.customCount} 自定义`];
    if (state.conflictCount > 0) parts.push(`${state.conflictCount} 组冲突`);
    return parts.join(' · ');
}

function skillHubSummaryLine(state: SkillHubState | null) {
    if (!state) return '浏览技能中心并安装到当前助手';
    return `技能中心 · ${state.total} 个可浏览技能`;
}

function formatCount(value: number) {
    if (!value) return '0';
    if (value >= 10000) return `${(value / 10000).toFixed(1)} 万`;
    return String(value);
}

function Badge(props: { label: string; tone: 'ok' | 'bad' | 'muted' }) {
    return <span className={`skill-badge ${props.tone}`}>{props.label}</span>;
}

function Meta(props: { label: string; value: string }) {
    return <div><dt>{props.label}</dt><dd>{props.value}</dd></div>;
}

function AuxiliaryModelsPanel(props: {
    model: ModelConfig;
    setModel: (value: ModelConfig) => void;
    providers: ProviderConfig;
    selectedAux: string;
    setSelectedAux: (value: string) => void;
    modelOptions: ModelOption[];
    auxModelOptions: Record<string, ModelOption[]>;
    auxModelListStatus: string;
    busy: boolean;
    onFetchAuxModels: (providerID: string) => void;
    onSave: () => Promise<boolean>;
    onBack: () => void;
}) {
    const enabledProviders = providerIDs(props.providers).filter((id) => !props.providers.providers[id].disabled);
    const selectedProviderID = props.providers.providers[props.model.provider] ? props.model.provider : firstProviderID(props.providers);
    const selectedProviderOptionsKey = modelOptionKey(selectedProviderID);
    const aux = props.model.auxiliary?.[props.selectedAux] || {provider: 'auto', model: '', baseUrl: '', apiKey: '', timeout: 30, extraBody: {}};
    const selectedAuxProviderID = aux.provider && aux.provider !== 'auto' && props.providers.providers[aux.provider] ? aux.provider : selectedProviderID;
    const selectedAuxProvider = props.providers.providers[selectedAuxProviderID];
    const auxProviderOptionsKey = modelOptionKey(selectedAuxProviderID);
    const auxUsesMainProvider = auxProviderOptionsKey === selectedProviderOptionsKey;
    const auxProviderOptions = props.auxModelOptions[auxProviderOptionsKey] || (auxUsesMainProvider ? props.modelOptions : []);
    const auxCurrentModel = aux.model || selectedAuxProvider?.defaultModel || props.model.default;
    const auxModelChoices = ensureCurrentModelOption(auxProviderOptions, auxCurrentModel);
    const auxProviderReady = !!selectedAuxProvider && !selectedAuxProvider.disabled && selectedAuxProvider.apiKey.trim() !== '';
    const customAuxiliary = props.model.auxiliaryMode === 'custom';

    const setAux = (next: AuxModel) => {
        props.setModel({...props.model, auxiliary: {...props.model.auxiliary, [props.selectedAux]: next}});
    };

    const setAuxiliaryMode = (mode: string) => {
        if (mode !== 'custom') {
            props.setModel({...props.model, auxiliaryMode: mode});
            return;
        }
        const initialized = {...props.model.auxiliary};
        for (const key of Object.keys(auxLabels)) {
            const current = initialized[key] || {provider: '', model: '', baseUrl: '', apiKey: '', timeout: 30, extraBody: {}};
            const useCurrentProvider = current.provider && current.provider !== 'auto';
            const currentProviderID = useCurrentProvider && props.providers.providers[current.provider] ? current.provider : selectedProviderID;
            initialized[key] = {
                ...current,
                provider: currentProviderID,
                model: current.model || props.model.default,
                baseUrl: '',
                apiKey: '',
                timeout: current.timeout || 30,
                extraBody: current.extraBody || {},
            };
        }
        props.setModel({...props.model, auxiliaryMode: mode, auxiliary: initialized});
    };

    const applyAuxProvider = (id: string) => {
        const provider = props.providers.providers[id];
        if (!provider) return;
        setAux({
            ...aux,
            provider: id,
            model: provider.defaultModel,
            baseUrl: '',
            apiKey: '',
            timeout: aux.timeout || 30,
            extraBody: aux.extraBody || {},
        });
    };

    const setAuxModel = (value: string) => {
        setAux({
            ...aux,
            provider: selectedAuxProviderID,
            model: value,
            baseUrl: '',
            apiKey: '',
            timeout: aux.timeout || 30,
            extraBody: aux.extraBody || {},
        });
    };

    return (
        <div className="advanced-model-panel">
            <div className="setup-card">
                <div className="section-head">
                    <div>
                        <p className="eyebrow">高级模型设置</p>
                        <h2>辅助模型</h2>
                        <p className="setup-subtitle">默认保持自动。只有需要分用途指定模型时，再调整这里。</p>
                    </div>
                </div>
                <div className="segmented">
                    {[
                        ['auto', '自动'],
                        ['follow-main', '跟随主模型'],
                        ['custom', '分别配置'],
                    ].map(([mode, label]) => (
                        <button key={mode} className={props.model.auxiliaryMode === mode ? 'selected' : ''} onClick={() => setAuxiliaryMode(mode)}>{label}</button>
                    ))}
                </div>
                {!customAuxiliary && (
                    <div className="mode-summary quiet">
                        <strong>{props.model.auxiliaryMode === 'follow-main' ? '使用主模型' : '由 Hermes 自动选择'}</strong>
                        <span>{props.model.auxiliaryMode === 'follow-main' ? props.model.default : '推荐给大多数助手使用'}</span>
                    </div>
                )}
                {customAuxiliary && (
                    <div className="aux-config-stack">
                        <label className="field">
                            <span>用途</span>
                            <select value={props.selectedAux} onChange={(event) => props.setSelectedAux(event.target.value)}>
                                {Object.keys(auxLabels).map((key) => <option key={key} value={key}>{auxLabels[key]}</option>)}
                            </select>
                        </label>
                        <label className="field">
                            <span>服务商</span>
                            <select value={selectedAuxProviderID} onChange={(event) => applyAuxProvider(event.target.value)}>
                                {enabledProviders.map((id) => {
                                    const provider = props.providers.providers[id];
                                    return <option key={id} value={id}>{provider.label}</option>;
                                })}
                            </select>
                        </label>
                        {selectedAuxProvider && selectedAuxProvider.apiKey.trim() === '' && <div className="form-warning">该供应商未配置 API 密钥。请先在基础模型服务里填写密钥。</div>}
                        <label className="field">
                            <span>模型</span>
                            {auxProviderOptions.length > 0 ? (
                                <select value={auxCurrentModel} onChange={(event) => setAuxModel(event.target.value)}>
                                    {auxCurrentModel.trim() === '' && <option value="">请选择模型</option>}
                                    {auxModelChoices.map((item) => <option key={item.id} value={item.id}>{item.ownedBy ? `${item.id} · ${item.ownedBy}` : item.id}</option>)}
                                </select>
                            ) : (
                                <input value={aux.model || ''} onChange={(event) => setAuxModel(event.target.value)}/>
                            )}
                        </label>
                        <div className="actions model-actions">
                            <button className="ghost" onClick={() => props.onFetchAuxModels(selectedAuxProviderID)} disabled={props.busy || !auxProviderReady}><RefreshCcw size={16}/>拉取模型列表</button>
                            {props.auxModelListStatus && <span className="inline-status">{props.auxModelListStatus}</span>}
                        </div>
                    </div>
                )}
                <div className="wizard-actions">
                    <button className="ghost" onClick={props.onBack} disabled={props.busy}><ChevronLeft size={16}/>返回摘要</button>
                    <button className="primary no-margin" onClick={props.onSave} disabled={props.busy}><Save size={16}/>保存高级模型设置</button>
                </div>
            </div>
        </div>
    );
}

function AssistantWizard(props: {
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
    onWeixinLogin: () => void;
    onCancelWeixin: () => void;
    onSaveWeCom: () => Promise<boolean>;
    onSaveFeishu: () => Promise<boolean>;
    onUnbindPlatform: (platform: PlatformKey) => void;
    onSaveCurrentPlatform: () => Promise<boolean>;
    onFinishSetup: (apply: boolean) => Promise<boolean>;
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
                        <CheckItem ok={props.hasPlatformBinding} label={props.hasPlatformBinding ? '已绑定至少一个平台' : '暂未绑定平台，此助手不会参与运行'}/>
                    </div>
                    <div className="actions">
                        <button className="ghost" onClick={() => previous && goToStep(previous)} disabled={props.busy}><ChevronLeft size={16}/>上一步</button>
                        {props.hasPlatformBinding ? (
                            <>
                                <button className="ghost" onClick={() => props.onFinishSetup(false)} disabled={props.busy || !modelReady}>仅完成，稍后应用</button>
                                <button className="primary no-margin" onClick={() => props.onFinishSetup(true)} disabled={props.busy || !modelReady}><RefreshCcw size={16}/>完成并应用重建</button>
                            </>
                        ) : (
                            <button className="primary no-margin" onClick={() => props.onFinishSetup(false)} disabled={props.busy || !modelReady}>完成</button>
                        )}
                    </div>
                </div>
            )}
        </div>
    );
}

function ModelServiceStep(props: {
    providers: ProviderConfig;
    setProviders: (value: ProviderConfig) => void;
    selectedProvider: string;
    setSelectedProvider: (value: string) => void;
    model: ModelConfig;
    setModel: (value: ModelConfig) => void;
    modelOptions: ModelOption[];
    modelListStatus: string;
    modelTestStatus: string;
    modelDirty: boolean;
    busy: boolean;
    showApiKey: boolean;
    setShowApiKey: (value: boolean) => void;
    onFetchModels: () => void;
    onTestModel: () => void;
    onSaveModelService: () => Promise<boolean>;
    stepLabel: string;
    stepHelp: string;
    onNext: () => void;
}) {
    const ids = providerIDs(props.providers);
    const selectedID = props.providers.providers[props.selectedProvider] ? props.selectedProvider : firstProviderID(props.providers);
    const selected = props.providers.providers[selectedID];
    const enabledProviders = ids.filter((id) => !props.providers.providers[id].disabled);
    const modelChoices = ensureCurrentModelOption(props.modelOptions, props.model.default);
    const modelReady = !!selected && props.model.default.trim() !== '';
    const modelCanTest = modelReady && !selected.disabled && selected.apiKey.trim() !== '';
    const selectedRefs = selectedID ? providerReferenceLabels(props.model, selectedID) : [];

    const updateProvider = (id: string, next: ProviderEntry) => {
        props.setProviders({providers: {...props.providers.providers, [id]: next}});
    };

    const applyProvider = (id: string) => {
        const provider = props.providers.providers[id];
        if (!provider) return;
        props.setSelectedProvider(id);
        props.setModel({...props.model, provider: id, default: props.model.provider === id ? props.model.default : provider.defaultModel});
    };

    const addProvider = () => {
        const id = nextProviderID(props.providers, '自定义供应商');
        const entry = {
            label: '自定义供应商',
            provider: 'custom',
            baseUrl: '',
            apiMode: 'chat_completions',
            apiKey: '',
            modelListUrl: '',
            defaultModel: '',
            builtin: false,
            disabled: false,
        };
        props.setProviders({
            providers: {
                ...props.providers.providers,
                [id]: entry,
            },
        });
        props.setSelectedProvider(id);
        props.setModel({...props.model, provider: id, default: entry.defaultModel});
    };

    const deleteSelectedProvider = () => {
        if (!selected || selected.builtin || ids.length <= 1) return;
        const nextProviders = {...props.providers.providers};
        delete nextProviders[selectedID];
        const nextConfig = {providers: nextProviders};
        const fallbackID = providerIDs(nextConfig).find((id) => !nextProviders[id].disabled) || firstProviderID(nextConfig);
        const fallback = nextProviders[fallbackID];
        const nextAuxiliary = {...props.model.auxiliary};
        for (const [key, aux] of Object.entries(nextAuxiliary)) {
            if (aux.provider === selectedID) {
                nextAuxiliary[key] = {
                    ...aux,
                    provider: fallbackID,
                    model: fallback?.defaultModel || props.model.default,
                    baseUrl: '',
                    apiKey: '',
                };
            }
        }
        props.setProviders(nextConfig);
        props.setSelectedProvider(fallbackID);
        props.setModel({
            ...props.model,
            provider: props.model.provider === selectedID ? fallbackID : props.model.provider,
            default: props.model.provider === selectedID ? (fallback?.defaultModel || '') : props.model.default,
            auxiliary: nextAuxiliary,
        });
    };

    return (
        <div className="wizard-panel">
            <div className="setup-card">
                <div className="section-head">
                    <div>
                        <p className="eyebrow">{props.stepLabel}</p>
                        <h2>连接模型服务</h2>
                        <p className="setup-subtitle">{props.stepHelp}</p>
                    </div>
                </div>
                {selected && (
                    <>
                        <label className="field">
                            <span>模型服务商</span>
                            <select value={selectedID} onChange={(event) => applyProvider(event.target.value)}>
                                {enabledProviders.map((id) => {
                                    const provider = props.providers.providers[id];
                                    return <option key={id} value={id}>{provider.label}</option>;
                                })}
                            </select>
                        </label>
                        <SecretField label="API 密钥" value={selected.apiKey} visible={props.showApiKey} setVisible={props.setShowApiKey} onChange={(value) => updateProvider(selectedID, {...selected, apiKey: value})}/>
                        <label className="field">
                            <span>模型</span>
                            {props.modelOptions.length > 0 ? (
                                <select value={props.model.default} onChange={(event) => props.setModel({...props.model, default: event.target.value})}>
                                    {props.model.default.trim() === '' && <option value="">请选择模型</option>}
                                    {modelChoices.map((item) => <option key={item.id} value={item.id}>{item.ownedBy ? `${item.id} · ${item.ownedBy}` : item.id}</option>)}
                                </select>
                            ) : (
                                <input value={props.model.default || ''} onChange={(event) => props.setModel({...props.model, default: event.target.value})}/>
                            )}
                        </label>
                        {selected.apiKey.trim() === '' && <div className="form-warning">API 密钥为空时可以保存选择，但不能测试或正常调用。</div>}
                        <div className="actions model-actions">
                            <button className="ghost" onClick={props.onFetchModels} disabled={props.busy || selected.apiKey.trim() === '' || selected.baseUrl.trim() === ''}><RefreshCcw size={16}/>拉取模型列表</button>
                            <button className="ghost" onClick={props.onTestModel} disabled={props.busy || !modelCanTest}><Activity size={16}/>{props.modelDirty ? '保存并测试模型' : '测试模型'}</button>
                            {props.modelListStatus && <span className="inline-status">{props.modelListStatus}</span>}
                            {props.modelTestStatus && <span className="inline-status">{props.modelTestStatus}</span>}
                        </div>
                        <details className="wizard-details">
                            <summary>其他选项</summary>
                            <div className="actions compact detail-provider-actions">
                                <button className="ghost detail-toggle" onClick={addProvider}>添加自定义服务</button>
                                {!selected.builtin && (
                                    <button className="ghost detail-toggle danger-inline" onClick={deleteSelectedProvider} disabled={props.busy || ids.length <= 1}>删除当前自定义服务</button>
                                )}
                            </div>
                            {!selected.builtin && selectedRefs.length > 0 && <div className="form-warning">删除时会把正在使用它的模型切回其他服务。</div>}
                            <div className="field-grid">
                                <Field label="显示名称" value={selected.label} onChange={(value) => updateProvider(selectedID, {...selected, label: value})}/>
                                <Field label="推荐默认模型" value={selected.defaultModel} onChange={(value) => updateProvider(selectedID, {...selected, defaultModel: value})}/>
                            </div>
                            <Field label="接口地址" value={selected.baseUrl} onChange={(value) => updateProvider(selectedID, {...selected, baseUrl: value})}/>
                            <label className="field">
                                <span>API 模式</span>
                                <select value={selected.apiMode || 'chat_completions'} onChange={(event) => updateProvider(selectedID, {...selected, apiMode: event.target.value})}>
                                    <option value="chat_completions">OpenAI Chat Completions</option>
                                    <option value="anthropic_messages">Anthropic Messages</option>
                                </select>
                            </label>
                            <Field label="模型列表地址" value={selected.modelListUrl} onChange={(value) => updateProvider(selectedID, {...selected, modelListUrl: value})}/>
                        </details>
                        <div className="setting-note">辅助模型保持自动策略，适合大多数新手场景；需要细调时再放到后续高级配置里处理。</div>
                    </>
                )}
                <div className="wizard-actions">
                    <button className="primary no-margin" onClick={async () => {
                        if (!(await props.onSaveModelService())) return;
                        props.onNext();
                    }} disabled={props.busy || !modelReady}>保存并下一步<ChevronRight size={16}/></button>
                </div>
            </div>
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

function assistantStatusLabel(setupCompletedAt?: string, status?: RuntimeProfileStatus, enabled = true, needsRebuild = false) {
    if (!setupCompletedAt) return '配置未完成';
    if (!enabled) return '已停用';
    if (needsRebuild) return '有未应用配置';
    return profileStatusText(status?.state, enabled);
}

function assistantStatusClass(setupCompletedAt?: string, status?: RuntimeProfileStatus, enabled = true, needsRebuild = false) {
    if (!setupCompletedAt || needsRebuild) return 'warn-status';
    return statusClassName(status?.state, enabled);
}

function wizardStepHelp(step: WizardStep) {
    switch (step) {
        case 'model':
            return '先选服务商、填密钥、确认主模型。';
        case 'soul':
            return '可以直接使用默认人格，也可以改成自己的助手说明。';
        case 'platforms':
            return '选择一个平台绑定；没有账号时可以暂不绑定。';
        case 'finish':
            return '确认配置结果，决定是否立即应用并重建。';
        default:
            return '';
    }
}

function skillSummaryLabel(state: SkillsState) {
    const base = `已安装 ${state.total} 个`;
    if (state.conflictCount > 0) return `${base}，冲突 ${state.conflictCount} 组`;
    return `${base}，内置 ${state.builtinCount} 个，自定义 ${state.customCount} 个`;
}

function formatBytes(value: number) {
    if (!value) return '0 B';
    if (value < 1024) return `${value} B`;
    if (value < 1024 * 1024) return `${(value / 1024).toFixed(1)} KB`;
    return `${(value / 1024 / 1024).toFixed(1)} MB`;
}

function suggestProfileID(profiles: Array<{ id: string }>, name: string) {
    const base = slugProfileID(name).replace(/-+$/, '') || 'assistant';
    const used = new Set(profiles.map((profile) => profile.id));
    let id = base;
    let index = 2;
    while (used.has(id) || id === 'default') {
        const suffix = `-${index}`;
        id = `${base.slice(0, 40 - suffix.length).replace(/-+$/, '')}${suffix}`;
        index += 1;
    }
    return id;
}
