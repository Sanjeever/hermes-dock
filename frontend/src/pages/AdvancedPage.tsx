import {useState} from 'react';
import {gotoLine, openSearchPanel} from '@codemirror/search';
import {EditorView} from '@codemirror/view';
import {CornerDownRight, Save, Search, Trash2} from 'lucide-react';
import {CodeEditor} from '../components/CodeEditor';

export function AdvancedPage(props: { options: Array<{ value: string; label: string }>; path: string; setPath: (value: string) => void; content: string; setContent: (value: string) => void; status: string; dirty: boolean; busy: boolean; onSave: () => void; onFactoryReset: () => Promise<void>; resetConfirmPhrase: string }) {
    const [editorView, setEditorView] = useState<EditorView | null>(null);
    const [resetConfirmText, setResetConfirmText] = useState('');
    const languageLabel = props.path.endsWith('.env') ? '.env' : 'YAML';
    const resetConfirmed = resetConfirmText === props.resetConfirmPhrase;

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
                        <p className="eyebrow">原始文件编辑器</p>
                        <h2>{props.path}</h2>
                    </div>
                    <span className={`inline-status ${props.dirty ? 'dirty' : ''}`}>{props.dirty ? '有未保存修改' : props.status}</span>
                </div>
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
                        <button className="primary" onClick={props.onSave} disabled={props.busy || !props.dirty}><Save size={16}/>保存</button>
                    </div>
                </div>
                <CodeEditor path={props.path} value={props.content} onChange={props.setContent} onReady={setEditorView}/>
            </div>
            <div className="panel danger-panel">
                <div className="section-head">
                    <div>
                        <p className="eyebrow">危险操作</p>
                        <h2>恢复出厂设置</h2>
                    </div>
                </div>
                <p className="muted">停止并移除 Hermes 容器，删除 ~/.hermes-dock，然后重新释放内置模板。该操作不可撤销。</p>
                <label className="reset-confirm">
                    <span>输入「{props.resetConfirmPhrase}」确认</span>
                    <input value={resetConfirmText} onChange={(event) => setResetConfirmText(event.target.value)} disabled={props.busy}/>
                </label>
                <button className="danger-button" onClick={factoryReset} disabled={props.busy || !resetConfirmed}><Trash2 size={16}/>恢复出厂设置</button>
            </div>
        </section>
    );
}
