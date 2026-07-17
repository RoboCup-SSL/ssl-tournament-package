// App bootstrap: router + Pinia + Quasar. No control/WS plugin (JSON only).
import { createApp } from 'vue'
import { createPinia } from 'pinia'
import { Quasar } from 'quasar'
import router from './router'
import App from './App.vue'

import '@quasar/extras/material-icons/material-icons.css'
import 'quasar/dist/quasar.css'
import '@/assets/main.scss'

createApp(App)
  .use(router)
  .use(createPinia())
  .use(Quasar, {})
  .mount('#app')
