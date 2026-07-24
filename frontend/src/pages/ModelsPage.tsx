import {Activity, RefreshCcw, Save} from 'lucide-react';
import {auxLabels} from '../constants';
import type {AuxModel, ModelConfig, ModelOption, ProviderConfig} from '../types';
import {ensureCurrentModelOption, firstProviderID, isVolcengineArkAgentPlanProvider, modelOptionKey, providerIDs} from '../utils';

export function ModelsPage(props: {
    model: ModelConfig;
    setModel: (value: ModelConfig) => void;
    selectedAux: string;
    setSelectedAux: (value: string) => void;
    providers: ProviderConfig;
    modelOptions: ModelOption[];
    modelListStatus: string;
    auxModelOptions: Record<string, ModelOption[]>;
    auxModelListStatus: string;
    busy: boolean;
    onFetchModels: () => void;
    onFetchAuxModels: (providerID: string) => void;
    onSave: () => void;
    onTest: () => void;
}) {
    const {model, setModel, selectedAux, setSelectedAux} = props;
    const aux = model.auxiliary?.[selectedAux] || {provider: 'auto', model: '', baseUrl: '', apiKey: '', timeout: 30, extraBody: {}};
    const setAux = (next: AuxModel) => setModel({...model, auxiliary: {...model.auxiliary, [selectedAux]: next}});
    const enabledProviders = providerIDs(props.providers).filter((id) => !props.providers.providers[id].disabled);
    const selectedProviderID = props.providers.providers[model.provider] ? model.provider : firstProviderID(props.providers);
    const selectedProvider = props.providers.providers[selectedProviderID];
    const selectedUsesBuiltinModelList = isVolcengineArkAgentPlanProvider(selectedProvider);
    const selectedProviderOptionsKey = modelOptionKey(selectedProviderID);
    const modelChoices = ensureCurrentModelOption(props.modelOptions, model.default);
    const showModelSelect = props.modelOptions.length > 0;
    const customAuxiliary = model.auxiliaryMode === 'custom';
    const modelReady = !!selectedProvider && model.default.trim() !== '';
    const modelCanTest = !!selectedProvider && modelReady && !selectedProvider.disabled && selectedProvider.apiKey.trim() !== '';
    const selectedAuxProviderID = aux.provider && aux.provider !== 'auto' && props.providers.providers[aux.provider] ? aux.provider : selectedProviderID;
    const selectedAuxProvider = props.providers.providers[selectedAuxProviderID];
    const auxProviderOptionsKey = modelOptionKey(selectedAuxProviderID);
    const auxUsesMainProvider = auxProviderOptionsKey === selectedProviderOptionsKey;
    const auxProviderOptions = props.auxModelOptions[auxProviderOptionsKey] || (auxUsesMainProvider ? props.modelOptions : []);
    const auxCurrentModel = aux.model || selectedAuxProvider?.defaultModel || model.default;
    const auxModelChoices = ensureCurrentModelOption(auxProviderOptions, auxCurrentModel);
    const showAuxModelSelect = auxProviderOptions.length > 0;
    const auxProviderReady = !!selectedAuxProvider && !selectedAuxProvider.disabled && (selectedAuxProvider.apiKey.trim() !== '' || isVolcengineArkAgentPlanProvider(selectedAuxProvider));
    const applyProvider = (id: string) => {
        const provider = props.providers.providers[id];
        if (!provider) return;
        setModel({
            ...model,
            provider: id,
            default: model.provider === id ? model.default : provider.defaultModel,
        });
    };
    const setAuxiliaryMode = (mode: string) => {
        if (mode !== 'custom') {
            setModel({...model, auxiliaryMode: mode});
            return;
        }
        const initialized = {...model.auxiliary};
        for (const key of Object.keys(auxLabels)) {
            const current = initialized[key] || {provider: '', model: '', baseUrl: '', apiKey: '', timeout: 30, extraBody: {}};
            const useCurrentProvider = current.provider && current.provider !== 'auto';
            const currentProviderID = useCurrentProvider && props.providers.providers[current.provider] ? current.provider : selectedProviderID;
            initialized[key] = {
                ...current,
                provider: currentProviderID,
                model: current.model || model.default,
                baseUrl: '',
                apiKey: '',
                timeout: current.timeout || 30,
                extraBody: current.extraBody || {},
            };
        }
        setModel({...model, auxiliaryMode: mode, auxiliary: initialized});
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
        <section className="grid two">
            <div className="panel">
                <div className="section-head">
                    <div>
                        <p className="eyebrow">主模型</p>
                        <h2>选择供应商和模型</h2>
                    </div>
                </div>
                <div className="provider-grid">
                    {enabledProviders.map((id) => {
                        const provider = props.providers.providers[id];
                        return (
                        <button key={id} className={`provider-card ${selectedProviderID === id ? 'selected' : ''}`} onClick={() => applyProvider(id)}>
                            <strong>{provider.label}</strong>
                            <span>{provider.apiKey.trim() === '' ? '未配置密钥' : provider.defaultModel || '手动填写模型'}</span>
                        </button>
                        );
                    })}
                </div>
                {selectedProvider?.disabled && <div className="form-warning">当前供应商已禁用，请重新选择或在供应商页启用。</div>}
                {selectedProvider && selectedProvider.apiKey.trim() === '' && <div className="form-warning">当前供应商未配置 API 密钥，可以保存模型选择，但不能测试或调用。</div>}
                <label className="field">
                    <span>模型</span>
                    {showModelSelect ? (
                        <select value={model.default} onChange={(event) => setModel({...model, default: event.target.value})}>
                            {model.default.trim() === '' && <option value="">请选择模型</option>}
                            {modelChoices.map((item) => <option key={item.id} value={item.id}>{item.ownedBy ? `${item.id} · ${item.ownedBy}` : item.id}</option>)}
                        </select>
                    ) : (
                        <input value={model.default || ''} onChange={(event) => setModel({...model, default: event.target.value})}/>
                    )}
                </label>
                <div className="actions model-actions">
                    <button className="ghost" onClick={props.onFetchModels} disabled={props.busy || !selectedProvider || selectedProvider.disabled || (!selectedUsesBuiltinModelList && selectedProvider.apiKey.trim() === '')}><RefreshCcw size={16}/>{selectedUsesBuiltinModelList ? '加载内置模型' : '拉取模型列表'}</button>
                    {props.modelListStatus && <span className="inline-status">{props.modelListStatus}</span>}
                </div>
                <div className="actions">
                    <button className="primary" onClick={props.onSave} disabled={props.busy || !modelReady}><Save size={16}/>保存模型配置</button>
                    <button className="ghost test-button" onClick={props.onTest} disabled={props.busy || !modelCanTest}><Activity size={16}/>测试模型</button>
                </div>
            </div>
            <div className="panel">
                <div className="section-head">
                    <div>
                        <p className="eyebrow">辅助模型</p>
                        <h2>选择策略</h2>
                    </div>
                </div>
                <div className="segmented">
                    {[
                        ['auto', '自动'],
                        ['follow-main', '跟随主模型'],
                        ['custom', '分别配置'],
                    ].map(([mode, label]) => (
                        <button key={mode} className={model.auxiliaryMode === mode ? 'selected' : ''} onClick={() => setAuxiliaryMode(mode)}>{label}</button>
                    ))}
                </div>
                {!customAuxiliary && (
                    <div className="mode-summary">
                        <strong>{model.auxiliaryMode === 'follow-main' ? '使用主模型' : '由 Hermes 自动选择'}</strong>
                        <span>{model.auxiliaryMode === 'follow-main' ? model.default : '适合大多数新手场景'}</span>
                    </div>
                )}
                {customAuxiliary && (
                    <>
                        <label className="field">
                            <span>用途</span>
                            <select value={selectedAux} onChange={(event) => setSelectedAux(event.target.value)}>
                                {Object.keys(auxLabels).map((key) => <option key={key} value={key}>{auxLabels[key]}</option>)}
                            </select>
                        </label>
                        <div className="provider-grid compact">
                            {enabledProviders.map((id) => {
                                const provider = props.providers.providers[id];
                                return (
                                <button key={id} className={`provider-card ${selectedAuxProviderID === id ? 'selected' : ''}`} onClick={() => applyAuxProvider(id)}>
                                    <strong>{provider.label}</strong>
                                    <span>{provider.apiKey.trim() === '' ? '未配置密钥' : provider.defaultModel || '手动填写模型'}</span>
                                </button>
                                );
                            })}
                        </div>
                        {selectedAuxProvider?.disabled && <div className="form-warning">该辅助模型供应商已禁用。</div>}
                        {selectedAuxProvider && selectedAuxProvider.apiKey.trim() === '' && <div className="form-warning">该供应商未配置 API 密钥。</div>}
                        <label className="field">
                            <span>模型</span>
                            {showAuxModelSelect ? (
                                <select value={auxCurrentModel} onChange={(event) => setAuxModel(event.target.value)}>
                                    {auxCurrentModel.trim() === '' && <option value="">请选择模型</option>}
                                    {auxModelChoices.map((item) => <option key={item.id} value={item.id}>{item.ownedBy ? `${item.id} · ${item.ownedBy}` : item.id}</option>)}
                                </select>
                            ) : (
                                <input value={aux.model || ''} onChange={(event) => setAuxModel(event.target.value)}/>
                            )}
                        </label>
                        <div className="actions model-actions">
                            <button className="ghost" onClick={() => props.onFetchAuxModels(selectedAuxProviderID)} disabled={props.busy || !auxProviderReady}><RefreshCcw size={16}/>{isVolcengineArkAgentPlanProvider(selectedAuxProvider) ? '加载内置模型' : '拉取模型列表'}</button>
                            {props.auxModelListStatus && <span className="inline-status">{props.auxModelListStatus}</span>}
                        </div>
                    </>
                )}
            </div>
        </section>
    );
}
