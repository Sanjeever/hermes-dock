import {useEffect, useRef, useState} from 'react';
import {Copy, RefreshCcw, X} from 'lucide-react';
import {ListProfileSkills} from '../services/api';
import type {
    BatchProfileConfigRequest,
    BatchProfileConfigResult,
    BundledContentSyncRequest,
    BundledContentSyncResult,
    ProfileEntry,
    SkillSummary,
    SkillsState,
} from '../types';

type ToolMode = 'copy' | 'sync';

export function ProfileBatchTools(props: {
    mode: ToolMode;
    profiles: ProfileEntry[];
    activeProfile: string;
    busy: boolean;
    onClose: () => void;
    onCopy: (request: BatchProfileConfigRequest) => Promise<BatchProfileConfigResult | null>;
    onSync: (request: BundledContentSyncRequest) => Promise<BundledContentSyncResult | null>;
}) {
    const dialogRef = useRef<HTMLElement | null>(null);
    const busyRef = useRef(props.busy);
    const [sourceID, setSourceID] = useState(props.activeProfile || props.profiles[0]?.id || 'default');
    const [targetIDs, setTargetIDs] = useState<string[]>(() => props.profiles.map((profile) => profile.id).filter((id) => props.mode === 'sync' || id !== props.activeProfile));
    const [copyMainModel, setCopyMainModel] = useState(true);
    const [copyAuxiliary, setCopyAuxiliary] = useState(true);
    const [copySoul, setCopySoul] = useState(false);
    const [copySkills, setCopySkills] = useState(false);
    const [copyProviders, setCopyProviders] = useState(true);
    const [includeAPIKeys, setIncludeAPIKeys] = useState(false);
    const [syncSoul, setSyncSoul] = useState(true);
    const [syncSkills, setSyncSkills] = useState(true);
    const [sourceSkills, setSourceSkills] = useState<SkillSummary[]>([]);
    const [skillPaths, setSkillPaths] = useState<string[]>([]);
    const [skillsLoading, setSkillsLoading] = useState(false);
    const [resultText, setResultText] = useState('');
    const [copyResult, setCopyResult] = useState<BatchProfileConfigResult | null>(null);
    const [syncResult, setSyncResult] = useState<BundledContentSyncResult | null>(null);

    useEffect(() => {
        busyRef.current = props.busy;
    }, [props.busy]);

    useEffect(() => {
        const previousFocus = document.activeElement instanceof HTMLElement ? document.activeElement : null;
        const dialog = dialogRef.current;
        const focusable = () => Array.from(dialog?.querySelectorAll<HTMLElement>('button:not([disabled]), input:not([disabled]), select:not([disabled])') || []);
        focusable()[0]?.focus();
        const handleKeyDown = (event: KeyboardEvent) => {
            if (event.key === 'Escape' && !busyRef.current) {
                event.preventDefault();
                props.onClose();
                return;
            }
            if (event.key !== 'Tab') return;
            const items = focusable();
            if (items.length === 0) return;
            const first = items[0];
            const last = items[items.length - 1];
            if (event.shiftKey && document.activeElement === first) {
                event.preventDefault();
                last.focus();
            } else if (!event.shiftKey && document.activeElement === last) {
                event.preventDefault();
                first.focus();
            }
        };
        document.addEventListener('keydown', handleKeyDown);
        return () => {
            document.removeEventListener('keydown', handleKeyDown);
            previousFocus?.focus();
        };
    }, []);

    useEffect(() => {
        if (props.mode !== 'copy' || !copySkills) return;
        let cancelled = false;
        setSkillsLoading(true);
        ListProfileSkills(sourceID).then((value) => {
            if (cancelled) return;
            const skills = (value as SkillsState).skills || [];
            setSourceSkills(skills);
            setSkillPaths((current) => current.filter((path) => skills.some((skill) => skill.path === path)));
        }).catch((error) => {
            if (!cancelled) setResultText(String(error));
        }).finally(() => {
            if (!cancelled) setSkillsLoading(false);
        });
        return () => {
            cancelled = true;
        };
    }, [props.mode, copySkills, sourceID]);

    const selectableTargets = props.profiles.filter((profile) => props.mode === 'sync' || profile.id !== sourceID);
    const allTargetsSelected = selectableTargets.length > 0 && selectableTargets.every((profile) => targetIDs.includes(profile.id));
    const toggleTarget = (id: string) => {
        setTargetIDs((current) => current.includes(id) ? current.filter((item) => item !== id) : [...current, id]);
    };
    const selectAllTargets = (checked: boolean) => {
        setTargetIDs(checked ? selectableTargets.map((profile) => profile.id) : []);
    };
    const selectSource = (id: string) => {
        setSourceID(id);
        setTargetIDs((current) => current.filter((target) => target !== id));
        setSkillPaths([]);
        setResultText('');
        setCopyResult(null);
    };
    const toggleSkill = (path: string) => {
        setSkillPaths((current) => current.includes(path) ? current.filter((item) => item !== path) : [...current, path]);
    };
    const copyReady = targetIDs.length > 0 && (copyMainModel || copyAuxiliary || copySoul || copyProviders || (copySkills && skillPaths.length > 0));
    const syncReady = targetIDs.length > 0 && (syncSoul || syncSkills);

    const submitCopy = async () => {
        if (includeAPIKeys && !window.confirm('API Key 属于敏感信息。确认复制到所选助手？')) return;
        const result = await props.onCopy({
            sourceProfileId: sourceID,
            targetProfileIds: targetIDs,
            copyMainModel,
            copyAuxiliary,
            copySoul,
            skillPaths: copySkills ? skillPaths : [],
            copyProviders,
            includeApiKeys: includeAPIKeys,
        });
        if (!result) return;
        setCopyResult(result);
        setSyncResult(null);
        setResultText(`已完成 ${result.succeeded} 个助手${result.failed ? `，${result.failed} 个失败` : ''}`);
    };

    const submitSync = async () => {
        const result = await props.onSync({targetProfileIds: targetIDs, syncSoul, syncSkills});
        if (!result) return;
        setSyncResult(result);
        setCopyResult(null);
        setResultText(`新增 ${result.added} 项，更新 ${result.updated} 项，保留 ${result.skipped} 项用户修改${result.failed ? `；${result.failed} 个助手失败` : ''}`);
    };

    return (
        <div className="assistant-create-overlay" role="presentation">
            <aside ref={dialogRef} className="assistant-create-drawer profile-batch-drawer" role="dialog" aria-modal="true" aria-labelledby="profile-batch-title">
                <div className="assistant-create-head">
                    <div>
                        <p className="eyebrow">多助手管理</p>
                        <h2 id="profile-batch-title">{props.mode === 'copy' ? '批量应用配置' : '同步内置内容'}</h2>
                    </div>
                    <button className="icon-button icon-only" type="button" onClick={props.onClose} disabled={props.busy} aria-label="关闭">
                        <X size={17}/>
                    </button>
                </div>
                <div className="profile-batch-body">
                    {props.mode === 'copy' && (
                        <label className="field">
                            <span>来源助手</span>
                            <select value={sourceID} onChange={(event) => selectSource(event.target.value)} disabled={props.busy}>
                                {props.profiles.map((profile) => <option key={profile.id} value={profile.id}>{profile.name || profile.id}</option>)}
                            </select>
                        </label>
                    )}

                    <fieldset className="batch-fieldset">
                        <legend>目标助手</legend>
                        <label className="mini-toggle"><input type="checkbox" checked={allTargetsSelected} onChange={(event) => selectAllTargets(event.target.checked)} disabled={props.busy}/>全选</label>
                        <div className="batch-target-list">
                            {selectableTargets.map((profile) => (
                                <label key={profile.id} className="batch-target">
                                    <input type="checkbox" checked={targetIDs.includes(profile.id)} onChange={() => toggleTarget(profile.id)} disabled={props.busy}/>
                                    <span>{profile.name || profile.id}</span>
                                    <code>{profile.id}</code>
                                </label>
                            ))}
                        </div>
                    </fieldset>

                    {props.mode === 'copy' ? (
                        <fieldset className="batch-fieldset">
                            <legend>复制内容</legend>
                            <div className="batch-options">
                                <label className="mini-toggle"><input type="checkbox" checked={copyMainModel} onChange={(event) => setCopyMainModel(event.target.checked)} disabled={props.busy}/>主模型</label>
                                <label className="mini-toggle"><input type="checkbox" checked={copyAuxiliary} onChange={(event) => setCopyAuxiliary(event.target.checked)} disabled={props.busy}/>辅助模型与策略</label>
                                <label className="mini-toggle"><input type="checkbox" checked={copySoul} onChange={(event) => setCopySoul(event.target.checked)} disabled={props.busy}/>人格 SOUL.md</label>
                                <label className="mini-toggle"><input type="checkbox" checked={copyProviders} onChange={(event) => {
                                    setCopyProviders(event.target.checked);
                                    if (!event.target.checked) setIncludeAPIKeys(false);
                                }} disabled={props.busy}/>供应商定义</label>
                                <label className="mini-toggle"><input type="checkbox" checked={copySkills} onChange={(event) => setCopySkills(event.target.checked)} disabled={props.busy}/>选择技能</label>
                                <label className="mini-toggle sensitive-option"><input type="checkbox" checked={includeAPIKeys} onChange={(event) => setIncludeAPIKeys(event.target.checked)} disabled={props.busy || !copyProviders}/>包含 API Key</label>
                            </div>
                            {copySkills && (
                                <div className="batch-skill-list">
                                    {skillsLoading ? <span>正在读取技能…</span> : sourceSkills.map((skill) => (
                                        <label key={skill.path} className="batch-target">
                                            <input type="checkbox" checked={skillPaths.includes(skill.path)} onChange={() => toggleSkill(skill.path)} disabled={props.busy}/>
                                            <span>{skill.name}</span>
                                            <code>{skill.builtin ? '内置' : '自定义'}</code>
                                        </label>
                                    ))}
                                    {!skillsLoading && sourceSkills.length === 0 && <span>来源助手暂无技能。</span>}
                                </div>
                            )}
                            <p className="field-hint">默认不复制 API Key；人格采用整份替换，技能只覆盖所选目录。</p>
                        </fieldset>
                    ) : (
                        <fieldset className="batch-fieldset">
                            <legend>同步内容</legend>
                            <div className="batch-options">
                                <label className="mini-toggle"><input type="checkbox" checked={syncSoul} onChange={(event) => setSyncSoul(event.target.checked)} disabled={props.busy}/>内置人格</label>
                                <label className="mini-toggle"><input type="checkbox" checked={syncSkills} onChange={(event) => setSyncSkills(event.target.checked)} disabled={props.busy}/>内置技能</label>
                            </div>
                            <p className="field-hint">内置人格会先备份再重置；内置技能只安全更新未修改内容，自定义技能和旧技能会保留。</p>
                        </fieldset>
                    )}

                    {resultText && (
                        <div className="batch-result" role="status">
                            <strong>{resultText}</strong>
                            {copyResult?.results.some((item) => !item.success) && (
                                <ul>
                                    {copyResult.results.filter((item) => !item.success).map((item, index) => (
                                        <li key={`${item.profileId}-${index}`}><code>{item.profileId || '未知助手'}</code><span>{item.error || '操作失败'}</span></li>
                                    ))}
                                </ul>
                            )}
                            {syncResult?.results.some((item) => !item.success || item.skipped > 0) && (
                                <ul>
                                    {syncResult.results.filter((item) => !item.success || item.skipped > 0).map((item, index) => (
                                        <li key={`${item.profileId}-${index}`}>
                                            <code>{item.profileId || '未知助手'}</code>
                                            <span>{item.success ? `保留 ${item.skipped} 项用户修改` : item.error || '同步失败'}</span>
                                        </li>
                                    ))}
                                </ul>
                            )}
                        </div>
                    )}
                </div>
                <div className="assistant-create-actions">
                    <button className="ghost" type="button" onClick={props.onClose} disabled={props.busy}>关闭</button>
                    <button className="primary no-margin" type="button" onClick={props.mode === 'copy' ? submitCopy : submitSync} disabled={props.busy || (props.mode === 'copy' ? !copyReady : !syncReady)}>
                        {props.mode === 'copy' ? <Copy size={16}/> : <RefreshCcw size={16}/>}
                        {props.mode === 'copy' ? `应用到 ${targetIDs.length} 个助手` : `同步到 ${targetIDs.length} 个助手`}
                    </button>
                </div>
            </aside>
        </div>
    );
}
