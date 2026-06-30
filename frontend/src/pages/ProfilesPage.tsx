import {useState} from 'react';
import {Save} from 'lucide-react';
import {Field} from '../components/fields';
import type {ProfileEntry, ProfileRegistry, RuntimeStatus} from '../types';
import {profileStatusText, slugProfileID, statusClassName} from '../utils';

export function ProfilesPage(props: {
    registry: ProfileRegistry;
    activeProfile: string;
    status: RuntimeStatus;
    busy: boolean;
    newProfileID: string;
    setNewProfileID: (value: string) => void;
    newProfileName: string;
    setNewProfileName: (value: string) => void;
    newProfileCopyMode: string;
    setNewProfileCopyMode: (value: string) => void;
    newProfileEnabled: boolean;
    setNewProfileEnabled: (value: boolean) => void;
    onSelect: (id: string) => void;
    onCreate: () => void;
    onRename: (id: string, name: string) => void;
    onEnabled: (id: string, enabled: boolean) => void;
    onMove: (id: string, direction: string) => void;
    onDelete: (id: string) => void;
}) {
    const profiles = props.registry?.profiles || [];
    const canCreate = /^[a-z0-9](?:[a-z0-9-]{0,38}[a-z0-9])$/.test(props.newProfileID) && props.newProfileID !== 'default';
    const [editingID, setEditingID] = useState('');
    const [editingName, setEditingName] = useState('');
    const [deleteID, setDeleteID] = useState('');
    const [deleteConfirmText, setDeleteConfirmText] = useState('');
    const startRename = (profile: ProfileEntry) => {
        setEditingID(profile.id);
        setEditingName(profile.name || profile.id);
    };
    const saveRename = (id: string) => {
        props.onRename(id, editingName);
        setEditingID('');
        setEditingName('');
    };
    return (
        <section className="grid two">
            <div className="panel">
                <div className="section-head">
                    <div>
                        <p className="eyebrow">Profiles</p>
                        <h2>助手列表</h2>
                    </div>
                </div>
                <div className="profile-list">
                    {profiles.map((profile, index) => {
                        const status = props.status?.profiles?.[profile.id];
                        const selected = props.activeProfile === profile.id;
                        return (
                            <div key={profile.id} className={`profile-row editable ${selected ? 'selected' : ''}`}>
                                <button className="profile-main" onClick={() => props.onSelect(profile.id)} disabled={props.busy}>
                                    <div>
                                        <strong>{profile.name || profile.id}</strong>
                                        <code>{profile.id}</code>
                                    </div>
                                    <span className={`profile-status ${statusClassName(status?.state, profile.enabled)}`}>{profileStatusText(status?.state, profile.enabled)}</span>
                                </button>
                                <div className="profile-controls">
                                    <label className="mini-toggle"><input type="checkbox" checked={profile.enabled} onChange={(event) => props.onEnabled(profile.id, event.target.checked)} disabled={props.busy}/>参与运行</label>
                                    <button className="ghost icon-only" title="上移" onClick={() => props.onMove(profile.id, 'up')} disabled={props.busy || index === 0}>↑</button>
                                    <button className="ghost icon-only" title="下移" onClick={() => props.onMove(profile.id, 'down')} disabled={props.busy || index === profiles.length - 1}>↓</button>
                                    <button className="ghost" onClick={() => startRename(profile)} disabled={props.busy || editingID === profile.id}>改名</button>
                                    <button className="ghost danger-inline" onClick={() => {
                                        setDeleteID(profile.id);
                                        setDeleteConfirmText('');
                                    }} disabled={props.busy || profile.id === 'default'}>删除</button>
                                </div>
                                {editingID === profile.id && (
                                    <div className="profile-rename">
                                        <input value={editingName} onChange={(event) => setEditingName(event.target.value)} autoFocus disabled={props.busy}/>
                                        <button className="primary inline-primary" onClick={() => saveRename(profile.id)} disabled={props.busy || editingName.trim() === ''}>保存</button>
                                        <button className="ghost" onClick={() => {
                                            setEditingID('');
                                            setEditingName('');
                                        }} disabled={props.busy}>取消</button>
                                    </div>
                                )}
                                {status?.message && <p className="profile-message">{status.message}</p>}
                                {deleteID === profile.id && (
                                    <div className="profile-rename danger-confirm">
                                        <input value={deleteConfirmText} onChange={(event) => setDeleteConfirmText(event.target.value)} placeholder={`输入 ${profile.id} 确认删除`} disabled={props.busy}/>
                                        <button className="danger-button compact" onClick={() => {
                                            props.onDelete(profile.id);
                                            setDeleteID('');
                                            setDeleteConfirmText('');
                                        }} disabled={props.busy || deleteConfirmText !== profile.id}>确认删除</button>
                                        <button className="ghost" onClick={() => {
                                            setDeleteID('');
                                            setDeleteConfirmText('');
                                        }} disabled={props.busy}>取消</button>
                                    </div>
                                )}
                            </div>
                        );
                    })}
                </div>
            </div>
            <div className="panel">
                <div className="section-head">
                    <div>
                        <p className="eyebrow">新建 Profile</p>
                        <h2>干净创建</h2>
                    </div>
                </div>
                <Field label="Profile ID" value={props.newProfileID} onChange={(value) => props.setNewProfileID(slugProfileID(value))}/>
                <Field label="显示名" value={props.newProfileName} onChange={props.setNewProfileName}/>
                <label className="field">
                    <span>创建方式</span>
                    <select value={props.newProfileCopyMode} onChange={(event) => props.setNewProfileCopyMode(event.target.value)}>
                        <option value="clean">干净 profile</option>
                        <option value="personality-skills">复制当前 profile 的人格和 skills</option>
                    </select>
                </label>
                <label className="mini-toggle profile-enable"><input type="checkbox" checked={props.newProfileEnabled} onChange={(event) => props.setNewProfileEnabled(event.target.checked)}/>创建后参与运行</label>
                <div className="setting-note">Profile ID 只能包含小写字母、数字和连字符，创建后不可修改。</div>
                {!canCreate && props.newProfileID && <div className="form-warning">Profile ID 格式不符合要求，或使用了保留 ID。</div>}
                <button className="primary" onClick={props.onCreate} disabled={props.busy || !canCreate}><Save size={16}/>创建 Profile</button>
            </div>
        </section>
    );
}
