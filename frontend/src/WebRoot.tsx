import {useEffect, useState} from 'react';
import type {FormEvent} from 'react';
import App from './App';
import {getWebSession, isWebRuntime, loginWeb, logoutWeb} from './services/api';

function WebRoot() {
    const [checking, setChecking] = useState(true);
    const [authenticated, setAuthenticated] = useState(false);
    const [password, setPassword] = useState('');
    const [error, setError] = useState('');

    useEffect(() => {
        if (!isWebRuntime()) {
            setAuthenticated(true);
            setChecking(false);
            return;
        }
        checkSession();
        const onExpired = () => setAuthenticated(false);
        window.addEventListener('web-session-expired', onExpired);
        return () => window.removeEventListener('web-session-expired', onExpired);
    }, []);

    async function checkSession() {
        setChecking(true);
        try {
            const session = await getWebSession();
            setAuthenticated(session.authenticated);
        } finally {
            setChecking(false);
        }
    }

    async function login(event: FormEvent) {
        event.preventDefault();
        setError('');
        try {
            await loginWeb(password);
            setPassword('');
            await checkSession();
        } catch (err) {
            setError(String(err));
        }
    }

    async function logout() {
        await logoutWeb();
        setAuthenticated(false);
    }

    if (!isWebRuntime()) return <App/>;
    if (checking) return <div className="web-login-screen"><div className="web-login-card">正在检查登录状态</div></div>;
    if (!authenticated) {
        return (
            <div className="web-login-screen">
                <form className="web-login-card" onSubmit={login}>
                    <div>
                        <p className="eyebrow">企智盒 Web 管理</p>
                        <h1>访问密码</h1>
                        <p>默认密码是 123456，建议首次登录后修改。</p>
                    </div>
                    <input type="hidden" name="username" value="hermes-dock" autoComplete="username" readOnly/>
                    <input type="password" value={password} onChange={(event) => setPassword(event.target.value)} autoComplete="current-password" autoFocus/>
                    {error && <div className="operation-error">{error}</div>}
                    <button className="primary no-margin" type="submit">登录</button>
                </form>
            </div>
        );
    }
    return (
        <div className="web-runtime-shell">
            <button className="web-logout-button" onClick={logout}>退出登录</button>
            <App/>
        </div>
    );
}

export default WebRoot;
