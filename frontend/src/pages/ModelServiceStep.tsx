import {Activity, ChevronRight, RefreshCcw, Server} from 'lucide-react';
import {Field, SecretField} from '../components/fields';
import type {ModelConfig, ModelOption, ProviderConfig, ProviderEntry} from '../types';
import {ensureCurrentModelOption, firstProviderID, isVolcengineArkAgentPlanProvider, providerIDs, providerReferenceLabels} from '../utils';

export function ModelServiceStep(props: {
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
    onOpenProviders: () => void;
}) {
    const ids = providerIDs(props.providers);
    const selectedID = props.providers.providers[props.selectedProvider] ? props.selectedProvider : firstProviderID(props.providers);
    const selected = props.providers.providers[selectedID];
    const usesBuiltinModelList = isVolcengineArkAgentPlanProvider(selected);
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
                        <SecretField label="API 密钥" value={selected.apiKey} visible={props.showApiKey} setVisible={props.setShowApiKey} onChange={(value) => updateProvider(selectedID, {...selected, apiKey: value})} hint="从供应商控制台获取"/>
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
                            <button className="ghost" onClick={props.onFetchModels} disabled={props.busy || (!usesBuiltinModelList && selected.apiKey.trim() === '') || selected.baseUrl.trim() === ''}><RefreshCcw size={16}/>{usesBuiltinModelList ? '加载内置模型' : '验证并拉取模型'}</button>
                            <button className="ghost" onClick={props.onTestModel} disabled={props.busy || !modelCanTest}><Activity size={16}/>{props.modelDirty ? '保存并测试模型' : '测试模型'}</button>
                            {props.modelListStatus && <span className="inline-status">{props.modelListStatus}</span>}
                            {props.modelTestStatus && <span className="inline-status">{props.modelTestStatus}</span>}
                        </div>
                        <details className="wizard-details">
                            <summary>其他选项</summary>
                            <div className="actions compact detail-provider-actions">
                                <button className="ghost detail-toggle" onClick={props.onOpenProviders}><Server size={16}/>供应商管理</button>
                                {!selected.builtin && (
                                    <button className="ghost detail-toggle danger-inline" onClick={deleteSelectedProvider} disabled={props.busy || ids.length <= 1}>删除当前自定义服务</button>
                                )}
                            </div>
                            {!selected.builtin && selectedRefs.length > 0 && <div className="form-warning">删除时会把正在使用它的模型切回其他服务。</div>}
                            <div className="field-grid">
                                <Field label="显示名称" value={selected.label} onChange={(value) => updateProvider(selectedID, {...selected, label: value})}/>
                                <Field label="推荐默认模型" value={selected.defaultModel} onChange={(value) => updateProvider(selectedID, {...selected, defaultModel: value})}/>
                            </div>
                            <Field label="接口地址" value={selected.baseUrl} onChange={(value) => updateProvider(selectedID, {...selected, baseUrl: value})} hint="OpenAI 兼容的 API 端点，如 https://api.example.com/v1"/>
                            <label className="field">
                                <span>API 模式</span>
                                <select value={selected.apiMode || 'chat_completions'} onChange={(event) => updateProvider(selectedID, {...selected, apiMode: event.target.value})}>
                                    <option value="chat_completions">OpenAI Chat Completions</option>
                                    <option value="anthropic_messages">Anthropic Messages</option>
                                </select>
                                <div className="field-hint">大多数供应商使用 OpenAI Chat Completions</div>
                            </label>
                            {usesBuiltinModelList ? (
                                <div className="setting-note">Agent Plan 使用 Hermes Dock 内置模型清单；同一密钥也用于图片、视频、豆包搜索和专业数据集。</div>
                            ) : (
                                <Field label="模型列表地址" value={selected.modelListUrl} onChange={(value) => updateProvider(selectedID, {...selected, modelListUrl: value})} hint="留空时自动使用接口地址 + /models"/>
                            )}
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
