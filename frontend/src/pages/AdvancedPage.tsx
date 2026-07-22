import {useState} from 'react';
import {gotoLine, openSearchPanel} from '@codemirror/search';
import {EditorView} from '@codemirror/view';
import {CornerDownRight, Download, FileCode2, FileSearch, Save, Search, Trash2, Upload} from 'lucide-react';
import {CodeEditor} from '../components/CodeEditor';
import {ConfirmDialog} from '../components/ConfirmDialog';
import {inspectedBackupMatchesInput} from '../backupPolicy';
import type {InstanceBackupManifest} from '../types';

export function AdvancedPage(props: { options: Array<{ value: string; label: string }>; path: string; setPath: (value: string) => void; open: boolean; setOpen: (value: boolean) => void; content: string; setContent: (value: string) => void; status: string; dirty: boolean; busy: boolean; webRuntime: boolean; backupStatus: string; backupManifest: InstanceBackupManifest | null; onExportBackup: (targetPath: string) => Promise<void>; onInspectBackup: (path: string) => Promise<void>; onImportBackup: (path: string, confirm: string) => Promise<void>; onClearBackupManifest: () => void; onSave: (confirm?: string) => void; onFactoryReset: () => Promise<void>; resetConfirmPhrase: string }) {
    const [editorView, setEditorView] = useState<EditorView | null>(null);
    const [resetConfirmText, setResetConfirmText] = useState('');
    const [exportPath, setExportPath] = useState('');
    const [importPath, setImportPath] = useState('');
    const [importConfirmText, setImportConfirmText] = useState('');
    const [pendingPath, setPendingPath] = useState('');
    const [composeSaveConfirmOpen, setComposeSaveConfirmOpen] = useState(false);
    const [composeSaveConfirmText, setComposeSaveConfirmText] = useState('');
    const languageLabel = props.path.endsWith('.env') ? '.env' : props.path.endsWith('.md') ? 'Markdown' : 'YAML';
    const resetConfirmed = resetConfirmText === props.resetConfirmPhrase;
    const importConfirmed = importConfirmText === '导入';
    const inspectedPath = props.backupManifest?.path || importPath;
    const importReady = inspectedBackupMatchesInput(props.backupManifest, importPath, props.webRuntime);
    const selectedOption = props.options.find((option) => option.value === props.path);

    async function factoryReset() {
        if (!resetConfirmed) return;
        await props.onFactoryReset();
        setResetConfirmText('');
    }

    function selectFile(path: string) {
        if (path === props.path) return true;
        if (props.dirty) {
            setPendingPath(path);
            return false;
        }
        props.setPath(path);
        return true;
    }

    function requestSave() {
        if (props.webRuntime && props.path === 'docker-compose.override.yaml') {
            setComposeSaveConfirmOpen(true);
            return;
        }
        props.onSave();
    }

    return (
        <section className="advanced-stack">
            <div className="panel">
                <div className="section-head">
                    <div>
                        <h2>{props.open ? selectedOption?.label || '配置文件' : '配置文件'}</h2>
                        {props.open ? <p className="setup-subtitle">{props.path}</p> : <p className="setup-subtitle">直接编辑原始配置文件。</p>}
                    </div>
                    {props.open && <span className={`inline-status ${props.dirty ? 'dirty' : ''}`}>{props.dirty ? '有未保存修改' : props.status}</span>}
                </div>
                {!props.open ? (
                    <div className="advanced-file-grid">
                        {props.options.map((option) => (
                            <button key={option.value} className="advanced-file-card" onClick={() => {
                                if (selectFile(option.value)) props.setOpen(true);
                            }}>
                                <FileCode2 size={20}/>
                                <strong>{option.label}</strong>
                                <span>{advancedFileHint(option.value)}</span>
                            </button>
                        ))}
                    </div>
                ) : (
                    <>
                        <div className="advanced-toolbar">
                            <select value={props.path} onChange={(event) => selectFile(event.target.value)}>
                                {props.options.map((option) => <option key={option.value} value={option.value}>{option.label}</option>)}
                            </select>
                            <div className="editor-actions">
                                <span className="language-badge">{languageLabel}</span>
                                <button type="button" className="ghost" onClick={() => editorView && openSearchPanel(editorView)} disabled={!editorView} title="搜索">
                                    <Search size={16}/>搜索
                                </button>
                                <button type="button" className="ghost" onClick={() => editorView && gotoLine(editorView)} disabled={!editorView} title="跳转到行">
                                    <CornerDownRight size={16}/>跳行
                                </button>
                                <button type="button" className="ghost" onClick={() => props.setOpen(false)} disabled={props.busy || props.dirty}>返回文件选择</button>
                                <button className="primary" onClick={requestSave} disabled={props.busy || !props.dirty}><Save size={16}/>保存</button>
                            </div>
                        </div>
                        <CodeEditor path={props.path} value={props.content} onChange={props.setContent} onReady={setEditorView}/>
                    </>
                )}
            </div>
            <ConfirmDialog
                open={!!pendingPath}
                title="放弃未保存修改？"
                description="切换配置文件会丢失当前编辑内容，此操作无法撤销。"
                confirmLabel="放弃修改并切换"
                tone="danger"
                busy={props.busy}
                onCancel={() => setPendingPath('')}
                onConfirm={() => {
                    const path = pendingPath;
                    setPendingPath('');
                    props.setPath(path);
                }}
            />
            <ConfirmDialog
                open={composeSaveConfirmOpen}
                title="保存 Compose 覆盖文件"
                description="错误配置可能导致服务无法启动。请输入“确认”后继续保存。"
                confirmLabel="确认保存"
                tone="danger"
                busy={props.busy}
                confirmDisabled={composeSaveConfirmText !== '确认'}
                onCancel={() => {
                    setComposeSaveConfirmOpen(false);
                    setComposeSaveConfirmText('');
                }}
                onConfirm={() => {
                    const confirm = composeSaveConfirmText;
                    setComposeSaveConfirmOpen(false);
                    setComposeSaveConfirmText('');
                    props.onSave(confirm);
                }}
            >
                <label>
                    <span>输入“确认”</span>
                    <input value={composeSaveConfirmText} onChange={(event) => setComposeSaveConfirmText(event.target.value)} disabled={props.busy} autoComplete="off"/>
                </label>
            </ConfirmDialog>
            <div className="panel backup-panel">
                <div className="section-head">
                    <div>
                        <h2>快速迁移备份</h2>
                        <p className="setup-subtitle">保留配置、凭据和业务数据，不包含可重建依赖、缓存及共享目录。</p>
                    </div>
                    {props.backupStatus && <span className="inline-status">{props.backupStatus}</span>}
                </div>
                <div className="backup-actions-grid">
                    <div className="backup-action">
                        <strong>导出当前实例</strong>
                        <span>自动跳过虚拟环境、依赖和缓存；导出时会临时停止容器，完成后自动恢复。</span>
                        {props.webRuntime && (
                            <label>
                                <span>服务器保存路径</span>
                                <input value={exportPath} onChange={(event) => setExportPath(event.target.value)} placeholder="/path/to/hermes-dock-backup.hdbackup" disabled={props.busy}/>
                            </label>
                        )}
                        <button className="primary no-margin" onClick={() => props.onExportBackup(exportPath)} disabled={props.busy || (props.webRuntime && exportPath.trim() === '')}><Download size={16}/>导出备份</button>
                    </div>
                    <div className="backup-action">
                        <strong>导入实例备份</strong>
                        <span>导入会替换当前实例，并先创建导入前备份。</span>
                        {props.webRuntime && (
                            <label>
                                <span>服务器备份路径</span>
                                <input value={importPath} onChange={(event) => {
                                    setImportPath(event.target.value);
                                    props.onClearBackupManifest();
                                }} placeholder="/path/to/hermes-dock-backup.hdbackup" disabled={props.busy}/>
                            </label>
                        )}
                        <div className="backup-button-row">
                            <button className="ghost" onClick={() => props.onInspectBackup(importPath)} disabled={props.busy || (props.webRuntime && importPath.trim() === '')}><FileSearch size={16}/>检查备份</button>
                            <button className="danger-button compact" onClick={() => props.onImportBackup(inspectedPath, importConfirmText)} disabled={props.busy || !importReady || !importConfirmed}><Upload size={16}/>导入备份</button>
                        </div>
                        <label>
                            <span>输入「导入」确认覆盖当前实例</span>
                            <input value={importConfirmText} onChange={(event) => setImportConfirmText(event.target.value)} disabled={props.busy || !importReady}/>
                        </label>
                    </div>
                </div>
                {props.backupManifest && (
                    <div className="backup-summary">
                        <div><span>创建时间</span><strong>{formatDateTime(props.backupManifest.createdAt)}</strong></div>
                        <div><span>来源版本</span><strong>{props.backupManifest.appVersion || '未知'}</strong></div>
                        <div><span>助手</span><strong>{props.backupManifest.profiles?.length || 0} 个</strong></div>
                        <div><span>文件</span><strong>{props.backupManifest.fileCount} 个，{formatBytes(props.backupManifest.totalBytes)}</strong></div>
                        <div><span>敏感信息</span><strong>{props.backupManifest.includesSecrets ? '包含密钥和凭据' : '不包含'}</strong></div>
                        <div><span>备份路径</span><strong>{props.backupManifest.path || '已选择'}</strong></div>
                    </div>
                )}
            </div>
            <details className="panel danger-panel">
                <summary className="danger-summary">
                    <span>
                        <span className="eyebrow">危险操作</span>
                        <strong>清除当前实例</strong>
                    </span>
                </summary>
                <div className="danger-body">
                    <p className="muted">保留共享目录，删除其他实例数据并恢复默认设置。此操作不可撤销。</p>
                    <label className="reset-confirm">
                        <span>输入「{props.resetConfirmPhrase}」确认</span>
                        <input value={resetConfirmText} onChange={(event) => setResetConfirmText(event.target.value)} disabled={props.busy}/>
                    </label>
                    <button className="danger-button" onClick={factoryReset} disabled={props.busy || !resetConfirmed}><Trash2 size={16}/>清除当前实例</button>
                </div>
            </details>
        </section>
    );
}

function advancedFileHint(path: string) {
    if (path.endsWith('.env')) return '密钥、平台变量和运行环境';
    if (path.endsWith('SOUL.md')) return '当前助手的人格设定';
    if (path.includes('override')) return '全局 Docker Compose 覆盖';
    return '当前助手的模型、终端和 Hermes 配置';
}

function formatBytes(value: number) {
    if (!Number.isFinite(value) || value <= 0) return '0 B';
    const units = ['B', 'KB', 'MB', 'GB'];
    let size = value;
    let index = 0;
    while (size >= 1024 && index < units.length - 1) {
        size /= 1024;
        index++;
    }
    return `${size >= 10 || index === 0 ? size.toFixed(0) : size.toFixed(1)} ${units[index]}`;
}

function formatDateTime(value: string) {
    if (!value) return '未知';
    const date = new Date(value);
    if (Number.isNaN(date.getTime())) return value;
    return date.toLocaleString('zh-CN', {hour12: false});
}
