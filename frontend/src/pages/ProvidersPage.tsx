import {ChevronLeft, RefreshCcw, Save, Trash2} from 'lucide-react';
import {Field, SecretField} from '../components/fields';
import type {ModelConfig, ModelOption, ProviderConfig, ProviderEntry} from '../types';
import {ensureCurrentModelOption, firstProviderID, nextProviderID, providerIDs, providerReferenceLabels} from '../utils';

export function ProvidersPage(props: {
    providers: ProviderConfig;
    setProviders: (value: ProviderConfig) => void;
    selectedProvider: string;
    setSelectedProvider: (value: string) => void;
    model: ModelConfig | null;
    busy: boolean;
    showApiKey: boolean;
    setShowApiKey: (value: boolean) => void;
    modelOptions: ModelOption[];
    modelListStatus: string;
    onFetchModels: (provider: ProviderEntry) => void;
    onSave: () => void;
    onBack?: () => void;
}) {
    const ids = providerIDs(props.providers);
    const selectedID = props.providers.providers[props.selectedProvider] ? props.selectedProvider : ids[0];
    const selected = props.providers.providers[selectedID];
    const refs = selectedID ? providerReferenceLabels(props.model, selectedID) : [];
    const updateSelected = (next: ProviderEntry) => {
        props.setProviders({providers: {...props.providers.providers, [selectedID]: next}});
    };
    const addProvider = () => {
        const id = nextProviderID(props.providers, '自定义供应商');
        props.setProviders({
            providers: {
                ...props.providers.providers,
                [id]: {
                    label: '自定义供应商',
                    provider: 'custom',
                    baseUrl: '',
                    apiMode: 'chat_completions',
                    apiKey: '',
                    modelListUrl: '',
                    defaultModel: '',
                    builtin: false,
                    disabled: false,
                },
            },
        });
        props.setSelectedProvider(id);
    };
    const deleteSelected = () => {
        if (!selected || selected.builtin || refs.length > 0) return;
        const next = {...props.providers.providers};
        delete next[selectedID];
        props.setProviders({providers: next});
        props.setSelectedProvider(firstProviderID({providers: next}));
    };
    if (!selected) {
        return (
            <section className="panel">
                <p className="eyebrow">供应商</p>
                <button className="primary" onClick={addProvider}>新增供应商</button>
            </section>
        );
    }
    return (
        <section className="grid two provider-layout">
            <div className="panel">
                {props.onBack && <button className="ghost" onClick={props.onBack} disabled={props.busy}><ChevronLeft size={16}/>返回摘要</button>}
                <div className="section-head">
                    <div>
                        <p className="eyebrow">供应商</p>
                        <h2>连接配置</h2>
                    </div>
                    <button className="ghost" onClick={addProvider}>新增</button>
                </div>
                <div className="provider-list">
                    {ids.map((id) => {
                        const item = props.providers.providers[id];
                        const configured = item.apiKey.trim() !== '';
                        return (
                            <button key={id} className={`provider-list-item ${id === selectedID ? 'selected' : ''}`} onClick={() => props.setSelectedProvider(id)}>
                                <strong>{item.label || id}</strong>
                                <span>{configured ? '已配置密钥' : '未配置密钥'} · {item.disabled ? '已禁用' : '启用中'}</span>
                            </button>
                        );
                    })}
                </div>
            </div>
            <div className="panel">
                <div className="section-head">
                    <div>
                        <p className="eyebrow">{selected.builtin ? '内置供应商' : '自定义供应商'}</p>
                        <h2>{selected.label || selectedID}</h2>
                    </div>
                    <label className="mini-toggle"><input type="checkbox" checked={!selected.disabled} onChange={(event) => updateSelected({...selected, disabled: !event.target.checked})}/>启用</label>
                </div>
                <div className="field-grid">
                    <Field label="显示名称" value={selected.label} onChange={(value) => updateSelected({...selected, label: value})}/>
                    <Field label="推荐默认模型" value={selected.defaultModel} onChange={(value) => updateSelected({...selected, defaultModel: value})}/>
                </div>
                <Field label="接口地址" value={selected.baseUrl} onChange={(value) => updateSelected({...selected, baseUrl: value})} hint="OpenAI 兼容的 API 端点，如 https://api.example.com/v1"/>
                <label className="field">
                    <span>API 模式</span>
                    <select value={selected.apiMode || 'chat_completions'} onChange={(event) => updateSelected({...selected, apiMode: event.target.value})}>
                        <option value="chat_completions">OpenAI Chat Completions</option>
                        <option value="anthropic_messages">Anthropic Messages</option>
                    </select>
                    <div className="field-hint">大多数供应商使用 OpenAI Chat Completions</div>
                </label>
                <SecretField label="API 密钥" value={selected.apiKey} visible={props.showApiKey} setVisible={props.setShowApiKey} onChange={(value) => updateSelected({...selected, apiKey: value})} hint="从供应商控制台获取"/>
                <Field label="模型列表地址" value={selected.modelListUrl} onChange={(value) => updateSelected({...selected, modelListUrl: value})} hint="留空时自动使用接口地址 + /models"/>
                {props.modelOptions.length > 0 && (
                    <label className="field">
                        <span>从已拉取模型中选择推荐默认模型</span>
                        <select value={selected.defaultModel} onChange={(event) => updateSelected({...selected, defaultModel: event.target.value})}>
                            {ensureCurrentModelOption(props.modelOptions, selected.defaultModel).map((item) => <option key={item.id} value={item.id}>{item.ownedBy ? `${item.id} · ${item.ownedBy}` : item.id}</option>)}
                        </select>
                    </label>
                )}
                {refs.length > 0 && <div className="form-warning">正在被使用：{refs.join('、')}</div>}
                <div className="actions">
                    <button className="ghost" onClick={() => props.onFetchModels(selected)} disabled={props.busy || selected.apiKey.trim() === '' || selected.baseUrl.trim() === ''}><RefreshCcw size={16}/>验证并拉取模型</button>
                    <button className="ghost" onClick={deleteSelected} disabled={props.busy || selected.builtin || refs.length > 0}><Trash2 size={16}/>删除</button>
                    <button className="primary" onClick={props.onSave} disabled={props.busy}><Save size={16}/>保存供应商配置</button>
                </div>
                {props.modelListStatus && <span className="inline-status">{props.modelListStatus}</span>}
            </div>
        </section>
    );
}
