
import { defineConfig } from 'vite'
import react from '@vitejs/plugin-react'
import tailwindcss from '@tailwindcss/vite'

export default defineConfig({
  base: './', // important for correct asset paths when served by Go
  plugins: [react(),tailwindcss(),]
})
