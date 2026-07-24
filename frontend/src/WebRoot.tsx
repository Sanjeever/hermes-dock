import {useEffect, useRef, useState} from 'react';
import type {FormEvent} from 'react';
import App from './App';
import {cancelWebRequests, getWebSession, isWebRuntime, loginWeb, logoutWeb} from './services/api';
import {disconnectEvents} from './services/events';

function WebRoot() {
    const [checking, setChecking] = useState(true);
    const [authenticated, setAuthenticated] = useState(false);
    const [password, setPassword] = useState('');
    const [error, setError] = useState('');
    const [sessionError, setSessionError] = useState('');
    const [loggingIn, setLoggingIn] = useState(false);
    const [loggingOut, setLoggingOut] = useState(false);
    const mounted = useRef(true);

    useEffect(() => {
        mounted.current = true;
        if (!isWebRuntime()) {
            setAuthenticated(true);
            setChecking(false);
            return () => { mounted.current = false; };
        }
        void checkSession();
        const onExpired = () => {
            cancelWebRequests();
            disconnectEvents();
            setAuthenticated(false);
            setError('登录已失效，请重新登录');
        };
        window.addEventListener('web-session-expired', onExpired);
        return () => {
            mounted.current = false;
            cancelWebRequests();
            disconnectEvents();
            window.removeEventListener('web-session-expired', onExpired);
        };
    }, []);

    async function checkSession() {
        setChecking(true);
        setSessionError('');
        try {
            const session = await getWebSession();
            if (!mounted.current) return;
            setAuthenticated(session.authenticated);
            if (!session.authenticated) {
                cancelWebRequests();
                disconnectEvents();
            }
        } catch (err) {
            if (!mounted.current) return;
            cancelWebRequests();
            disconnectEvents();
            setAuthenticated(false);
            setSessionError(errorMessage(err));
        } finally {
            if (mounted.current) setChecking(false);
        }
    }

    async function login(event: FormEvent) {
        event.preventDefault();
        setError('');
        setLoggingIn(true);
        try {
            await loginWeb(password);
            if (!mounted.current) return;
            setPassword('');
            await checkSession();
        } catch (err) {
            if (mounted.current) setError(errorMessage(err));
        } finally {
            if (mounted.current) setLoggingIn(false);
        }
    }

    async function logout() {
        setError('');
        setLoggingOut(true);
        try {
            await logoutWeb();
            if (!mounted.current) return;
            cancelWebRequests();
            disconnectEvents();
            setAuthenticated(false);
        } catch (err) {
            if (mounted.current) setError(errorMessage(err));
        } finally {
            if (mounted.current) setLoggingOut(false);
        }
    }

    if (!isWebRuntime()) return <App/>;
    if (checking) return <div className="web-login-screen"><div className="web-login-card">正在检查登录状态</div></div>;
    if (sessionError) {
        return (
            <div className="web-login-screen">
                <div className="web-login-card">
                    <div>
                        <p className="eyebrow">企智盒 Web 管理</p>
                        <h1>暂时无法检查登录状态</h1>
                        <p>请确认桌面端仍在运行，并检查当前网络连接。</p>
                    </div>
                    <div className="operation-error">{sessionError}</div>
                    <button className="primary no-margin" type="button" onClick={() => void checkSession()}>重新检查</button>
                </div>
            </div>
        );
    }
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
                    <input type="password" value={password} onChange={(event) => setPassword(event.target.value)} autoComplete="current-password" autoFocus disabled={loggingIn}/>
                    {error && <div className="operation-error">{error}</div>}
                    <button className="primary no-margin" type="submit" disabled={loggingIn || password === ''}>{loggingIn ? '正在登录' : '登录'}</button>
                </form>
            </div>
        );
    }
    return (
        <div className="web-runtime-shell">
            {error && <div className="operation-error web-logout-error">{error}</div>}
            <button className="web-logout-button" onClick={() => void logout()} disabled={loggingOut}>{loggingOut ? '正在退出' : '退出登录'}</button>
            <App/>
        </div>
    );
}

function errorMessage(error: unknown) {
    return error instanceof Error ? error.message : String(error);
}

export default WebRoot;
