import {describe, expect, it} from 'vitest';
import {
    defaultAdvancedPath,
    isPortValue,
    providerReferenceLabels,
    setEnvValue,
    slugProfileID,
    toPlainModelConfig,
} from './utils';
import type {ModelConfig} from './types';

describe('profile utilities', () => {
    it('normalizes profile IDs and resolves profile config paths', () => {
        expect(slugProfileID(' Sales / 华东 ')).toBe('sales-');
        expect(defaultAdvancedPath('default')).toBe('data/config.yaml');
        expect(defaultAdvancedPath('sales')).toBe('data/profiles/sales/config.yaml');
    });
});

describe('deployment utilities', () => {
    it('accepts only valid TCP port numbers', () => {
        expect(isPortValue('1')).toBe(true);
        expect(isPortValue('65535')).toBe(true);
        expect(isPortValue('0')).toBe(false);
        expect(isPortValue('65536')).toBe(false);
        expect(isPortValue('12.5')).toBe(false);
    });
});

describe('configuration utilities', () => {
    it('updates existing environment values without mutating the input', () => {
        const input = [{key: 'TOKEN', value: 'old', secret: true}];
        const result = setEnvValue(input, 'TOKEN', 'new');
        expect(result).toEqual([{key: 'TOKEN', value: 'new', secret: true}]);
        expect(input[0].value).toBe('old');
    });

    it('reports every model reference to a provider', () => {
        const model: ModelConfig = {
            provider: 'deepseek',
            default: 'main',
            baseUrl: '',
            apiMode: '',
            apiKey: '',
            auxiliaryMode: 'custom',
            auxiliary: {
                vision: {provider: 'deepseek', model: 'vision', baseUrl: '', apiKey: '', timeout: 30, extraBody: {}},
            },
        };
        expect(providerReferenceLabels(model, 'deepseek')).toEqual(['主模型', '辅助模型：视觉理解']);
    });

    it('removes expanded secrets from the model save payload', () => {
        const model: ModelConfig = {
            provider: 'deepseek',
            default: 'main',
            baseUrl: 'https://example.com',
            apiMode: 'openai',
            apiKey: 'main-secret',
            auxiliaryMode: 'custom',
            auxiliary: {
                vision: {provider: 'deepseek', model: 'vision', baseUrl: 'https://example.com', apiKey: 'aux-secret', timeout: 30, extraBody: {}},
            },
        };
        const result = toPlainModelConfig(model);
        expect(result.apiKey).toBe('');
        expect(result.baseUrl).toBe('');
        expect(result.auxiliary.vision.apiKey).toBe('');
        expect(model.apiKey).toBe('main-secret');
    });
});
