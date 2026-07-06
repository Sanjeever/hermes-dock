import React from 'react'
import {createRoot} from 'react-dom/client'
import './style.css'
import WebRoot from './WebRoot'

const container = document.getElementById('root')

const root = createRoot(container!)

root.render(
    <React.StrictMode>
        <WebRoot/>
    </React.StrictMode>
)
