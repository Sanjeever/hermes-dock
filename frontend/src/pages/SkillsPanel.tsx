import {useEffect, useState} from 'react';
import {CheckSquare2, ChevronLeft, Download, FolderOpen, ListChecks, RefreshCcw, RotateCcw, Search, Square, Trash2} from 'lucide-react';
import type {SkillDetail, SkillHubDetail, SkillHubQuery, SkillHubState, SkillsState} from '../types';
import {formatBytes} from './assistantUtils';

export function SkillsPanel(props: {
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
    onRestoreDefaultSkills: () => Promise<boolean>;
    onDetail: (path: string) => void;
    onDelete: (path: string) => Promise<boolean>;
    onDeleteMany: (paths: string[]) => Promise<boolean>;
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
    const [batchMode, setBatchMode] = useState(false);
    const [selectedPaths, setSelectedPaths] = useState<Set<string>>(new Set());
    const [batchDeleteOpen, setBatchDeleteOpen] = useState(false);
    const [restoreDefaultOpen, setRestoreDefaultOpen] = useState(false);
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
    const allPaths = skills.map((skill) => skill.path).join('\n');
    const selectedCount = selectedPaths.size;
    const allFilteredSelected = filtered.length > 0 && filtered.every((skill) => selectedPaths.has(skill.path));
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
        setBatchMode(false);
        setSelectedPaths(new Set());
        setBatchDeleteOpen(false);
    }, [props.skillsState?.activeProfile]);

    useEffect(() => {
        const installed = new Set(skills.map((skill) => skill.path));
        setSelectedPaths((current) => {
            const next = new Set([...current].filter((path) => installed.has(path)));
            return next.size === current.size ? current : next;
        });
    }, [allPaths]);

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
    const leaveBatchMode = () => {
        setBatchMode(false);
        setSelectedPaths(new Set());
        setBatchDeleteOpen(false);
    };
    const toggleSelectedPath = (path: string) => {
        setSelectedPaths((current) => {
            const next = new Set(current);
            if (next.has(path)) next.delete(path);
            else next.add(path);
            return next;
        });
        setBatchDeleteOpen(false);
    };

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
                        <button className={view === 'hub' ? 'selected' : ''} onClick={() => {
                            setView('hub');
                            leaveBatchMode();
                        }} disabled={props.busy}>技能中心</button>
                    </div>
                    <div className="skills-head-actions">
                        {view === 'local' && <button className="ghost" onClick={props.onSyncBundledSkills} disabled={props.busy || batchMode}><Download size={16}/>同步内置技能</button>}
                        {view === 'local' && <button className="ghost danger-inline" onClick={() => setRestoreDefaultOpen(true)} disabled={props.busy || batchMode}><RotateCcw size={16}/>恢复默认技能</button>}
                        {view === 'local' && !batchMode && <button className="ghost danger-inline" onClick={() => {
                            setRestoreDefaultOpen(false);
                            setBatchMode(true);
                        }} disabled={props.busy || skills.length === 0}><ListChecks size={16}/>批量删除</button>}
                        {view === 'local' && batchMode && <button className="ghost" onClick={leaveBatchMode} disabled={props.busy}>取消批量</button>}
                        <button className="ghost" onClick={() => view === 'local' ? props.onRefresh() : searchHub(hubKeyword, hubCategory, hubPage)} disabled={props.busy}><RefreshCcw size={16}/>刷新</button>
                        <button className="ghost" onClick={props.onBack} disabled={props.busy}><ChevronLeft size={16}/>返回摘要</button>
                    </div>
                </div>
                <div className="skills-controls">
                    {view === 'local' && restoreDefaultOpen && (
                        <div className="danger-confirm restore-confirm-row">
                            <span>将删除当前助手的全部技能，并恢复为内置默认技能。操作前会自动备份。</span>
                            <button className="danger-button compact" onClick={async () => {
                                if (await props.onRestoreDefaultSkills()) setRestoreDefaultOpen(false);
                            }} disabled={props.busy}><RotateCcw size={16}/>确认恢复</button>
                            <button className="ghost" onClick={() => setRestoreDefaultOpen(false)} disabled={props.busy}>取消</button>
                        </div>
                    )}
                    {view === 'local' ? (
                        <>
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
                            {batchMode && (
                                <div className="skill-batch-toolbar">
                                    <span>已选 {selectedCount} 个技能</span>
                                    <button className="ghost" onClick={() => setSelectedPaths((current) => {
                                        const next = new Set(current);
                                        filtered.forEach((skill) => next.add(skill.path));
                                        return next;
                                    })} disabled={props.busy || allFilteredSelected}>选择当前结果</button>
                                    <button className="ghost" onClick={() => {
                                        setSelectedPaths(new Set());
                                        setBatchDeleteOpen(false);
                                    }} disabled={props.busy || selectedCount === 0}>清空</button>
                                    <button className="danger-button compact" onClick={() => setBatchDeleteOpen(true)} disabled={props.busy || selectedCount === 0}><Trash2 size={16}/>删除所选</button>
                                </div>
                            )}
                            {batchMode && batchDeleteOpen && (
                                <div className="danger-confirm skill-batch-confirm">
                                    <span>确认删除选中的 {selectedCount} 个技能？删除前会逐个备份，重建后生效。</span>
                                    <button className="danger-button compact" onClick={async () => {
                                        if (await props.onDeleteMany([...selectedPaths])) leaveBatchMode();
                                    }} disabled={props.busy || selectedCount === 0}><Trash2 size={16}/>确认批量删除</button>
                                    <button className="ghost" onClick={() => setBatchDeleteOpen(false)} disabled={props.busy}>取消</button>
                                </div>
                            )}
                        </>
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
                                        <button key={skill.path} aria-pressed={batchMode ? selectedPaths.has(skill.path) : undefined} className={`skill-row ${skill.conflict ? 'conflict' : ''} ${activeDetail?.path === skill.path && !batchMode ? 'selected' : ''} ${batchMode ? 'batch' : ''} ${selectedPaths.has(skill.path) ? 'batch-selected' : ''}`} onClick={() => {
                                            if (batchMode) {
                                                toggleSelectedPath(skill.path);
                                                return;
                                            }
                                            setDeletePath('');
                                            setDetailTab('overview');
                                            props.onDetail(skill.path);
                                        }}>
                                            {batchMode && <span className="skill-select-icon" aria-hidden="true">{selectedPaths.has(skill.path) ? <CheckSquare2 size={18}/> : <Square size={18}/>}</span>}
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
