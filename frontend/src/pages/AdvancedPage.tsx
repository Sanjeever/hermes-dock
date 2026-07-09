import {useState} from 'react';
import {gotoLine, openSearchPanel} from '@codemirror/search';
import {EditorView} from '@codemirror/view';
import {CornerDownRight, Download, FileCode2, FileSearch, Save, Search, Trash2, Upload} from 'lucide-react';
import {CodeEditor} from '../components/CodeEditor';
import type {InstanceBackupManifest} from '../types';

export function AdvancedPage(props: { options: Array<{ value: string; label: string }>; path: string; setPath: (value: string) => void; open: boolean; setOpen: (value: boolean) => void; content: string; setContent: (value: string) => void; status: string; dirty: boolean; busy: boolean; webRuntime: boolean; backupStatus: string; backupManifest: InstanceBackupManifest | null; onExportBackup: (targetPath: string) => Promise<void>; onInspectBackup: (path: string) => Promise<void>; onImportBackup: (path: string, confirm: string) => Promise<void>; onSave: () => void; onFactoryReset: () => Promise<void>; resetConfirmPhrase: string }) {
    const [editorView, setEditorView] = useState<EditorView | null>(null);
    const [resetConfirmText, setResetConfirmText] = useState('');
    const [exportPath, setExportPath] = useState('');
    const [importPath, setImportPath] = useState('');
    const [importConfirmText, setImportConfirmText] = useState('');
    const languageLabel = props.path.endsWith('.env') ? '.env' : props.path.endsWith('.md') ? 'Markdown' : 'YAML';
    const resetConfirmed = resetConfirmText === props.resetConfirmPhrase;
    const importConfirmed = importConfirmText === '导入';
    const inspectedPath = props.backupManifest?.path || importPath;

    async function factoryReset() {
        if (!resetConfirmed) return;
        await props.onFactoryReset();
        setResetConfirmText('');
    }

    return (
        <section className="advanced-stack">
            <div className="panel">
                <div className="section-head">
                    <div>
                        <p className="eyebrow">高级编辑</p>
                        <h2>{props.open ? props.path : '选择要编辑的文件'}</h2>
                        {!props.open && <p className="setup-subtitle">直接修改原始配置文件。保存后通常需要应用配置才会生效。</p>}
                    </div>
                    {props.open && <span className={`inline-status ${props.dirty ? 'dirty' : ''}`}>{props.dirty ? '有未保存修改' : props.status}</span>}
                </div>
                {!props.open ? (
                    <div className="advanced-file-grid">
                        {props.options.map((option) => (
                            <button key={option.value} className="advanced-file-card" onClick={() => {
                                props.setPath(option.value);
                                props.setOpen(true);
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
                            <select value={props.path} onChange={(event) => props.setPath(event.target.value)}>
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
                                <button className="primary" onClick={props.onSave} disabled={props.busy || !props.dirty}><Save size={16}/>保存</button>
                            </div>
                        </div>
                        <CodeEditor path={props.path} value={props.content} onChange={props.setContent} onReady={setEditorView}/>
                    </>
                )}
            </div>
            <div className="panel backup-panel">
                <div className="section-head">
                    <div>
                        <p className="eyebrow">数据迁移</p>
                        <h2>导出和导入实例备份</h2>
                        <p className="setup-subtitle">备份包含 profile、人格、技能、模型配置、平台绑定、账号凭据和密钥。</p>
                    </div>
                    {props.backupStatus && <span className="inline-status">{props.backupStatus}</span>}
                </div>
                <div className="backup-actions-grid">
                    <div className="backup-action">
                        <strong>导出当前实例</strong>
                        <span>导出前会临时停止正在运行的容器，完成后恢复；运行日志、Web 会话和派生运行态不会写入备份。</span>
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
                        <span>导入会整体替换当前实例；开始前会自动保存当前设备的导入前备份。</span>
                        {props.webRuntime && (
                            <label>
                                <span>服务器备份路径</span>
                                <input value={importPath} onChange={(event) => setImportPath(event.target.value)} placeholder="/path/to/hermes-dock-backup.hdbackup" disabled={props.busy}/>
                            </label>
                        )}
                        <div className="backup-button-row">
                            <button className="ghost" onClick={() => props.onInspectBackup(importPath)} disabled={props.busy || (props.webRuntime && importPath.trim() === '')}><FileSearch size={16}/>检查备份</button>
                            <button className="danger-button compact" onClick={() => props.onImportBackup(inspectedPath, importConfirmText)} disabled={props.busy || !props.backupManifest || !importConfirmed}><Upload size={16}/>导入备份</button>
                        </div>
                        <label>
                            <span>输入「导入」确认覆盖当前实例</span>
                            <input value={importConfirmText} onChange={(event) => setImportConfirmText(event.target.value)} disabled={props.busy || !props.backupManifest}/>
                        </label>
                    </div>
                </div>
                {props.backupManifest && (
                    <div className="backup-summary">
                        <div><span>创建时间</span><strong>{formatDateTime(props.backupManifest.createdAt)}</strong></div>
                        <div><span>来源版本</span><strong>{props.backupManifest.appVersion || '未知'}</strong></div>
                        <div><span>Profiles</span><strong>{props.backupManifest.profiles?.length || 0} 个</strong></div>
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
                        <strong>恢复出厂设置</strong>
                    </span>
                </summary>
                <div className="danger-body">
                    <p className="muted">停止并移除 Hermes 容器，删除 ~/.hermes-dock，然后重新释放内置模板。该操作不可撤销。</p>
                    <label className="reset-confirm">
                        <span>输入「{props.resetConfirmPhrase}」确认</span>
                        <input value={resetConfirmText} onChange={(event) => setResetConfirmText(event.target.value)} disabled={props.busy}/>
                    </label>
                    <button className="danger-button" onClick={factoryReset} disabled={props.busy || !resetConfirmed}><Trash2 size={16}/>恢复出厂设置</button>
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
