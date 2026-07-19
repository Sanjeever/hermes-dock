import {StrictMode} from 'react';
import {createRoot, hydrateRoot} from 'react-dom/client';
import ManualPage from './ManualPage';
import './manual.css';

const container = document.getElementById('root')!;
const app = (
    <StrictMode>
        <ManualPage />
    </StrictMode>
);

if (container.hasChildNodes()) {
    hydrateRoot(container, app);
} else {
    createRoot(container).render(app);
}
