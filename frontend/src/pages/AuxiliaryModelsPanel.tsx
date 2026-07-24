import {ChevronLeft, RefreshCcw, Save} from 'lucide-react';
import {auxLabels} from '../constants';
import type {AuxModel, ModelConfig, ModelOption, ProviderConfig} from '../types';
import {ensureCurrentModelOption, firstProviderID, isVolcengineArkAgentPlanProvider, modelOptionKey, providerIDs} from '../utils';

export function AuxiliaryModelsPanel(props: {
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
    const auxProviderReady = !!selectedAuxProvider && !selectedAuxProvider.disabled && (selectedAuxProvider.apiKey.trim() !== '' || isVolcengineArkAgentPlanProvider(selectedAuxProvider));
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
                            <button className="ghost" onClick={() => props.onFetchAuxModels(selectedAuxProviderID)} disabled={props.busy || !auxProviderReady}><RefreshCcw size={16}/>{isVolcengineArkAgentPlanProvider(selectedAuxProvider) ? '加载内置模型' : '拉取模型列表'}</button>
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
