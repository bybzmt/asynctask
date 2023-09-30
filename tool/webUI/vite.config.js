import { defineConfig } from 'vite'
import { svelte } from '@sveltejs/vite-plugin-svelte';
import tailwindcss from 'tailwindcss';
import path from 'path';

import { fileURLToPath } from 'url';
const __dirname = path.dirname(fileURLToPath(import.meta.url));

let dir_src = path.resolve(__dirname, './src');

export default defineConfig(({ command, mode }) => {
    console.log("command", command, "mode", mode)

    let api_base = JSON.stringify("");

    let postcss_config = {
        plugins: [
            tailwindcss({
                mode: 'jit',
                enabled: true,
                content: ["src/**/*.svelte"]
            }),
        ],
    }

    return {
        publicDir: path.resolve(__dirname, './static'),
        base: '/',
        root: './src',
        define: {
            API_BASE: api_base,
        },
        build: {
            emptyOutDir: false,
            sourcemap: true,
            cssCodeSplit: false,
        },
        resolve: {
            alias: {
                $src: dir_src,
            }
        },
        css: {
            postcss: postcss_config,
            preprocessorOptions: {},
        },
        plugins: [
            svelte({
                configFile: false,
                compilerOptions: {
                    hydratable: true,
                },
                disableDependencyReinclusion: true,
                extensions: [".svelte"],
                //useVitePreprocess:true,
                onwarn: (warning, handler) => {
                    if (/^a11y-/.test(warning.code)) return
                    handler(warning)
                }
            }),
        ],
        server: {
            host: "0.0.0.0",
            port: 3000,
        }
    }
})
