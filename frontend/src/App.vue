<script setup lang="ts">
// App chrome: header with the server version and a link to the preserved admin.
import { onMounted, ref } from 'vue'
import { api } from '@/api/client'
import type { Version } from '@/api/types'

const version = ref('')

onMounted(async () => {
  try {
    version.value = (await api.get<Version>('/api/version')).version
  } catch {
    version.value = ''
  }
})
</script>

<template>
  <q-layout view="hHh lpR fFf">
    <q-header elevated>
      <q-toolbar>
        <q-toolbar-title>
          SSL Tournament <span class="text-caption">{{ version }}</span>
        </q-toolbar-title>
        <q-btn flat dense no-caps label="admin" type="a" href="/admin/" />
      </q-toolbar>
    </q-header>
    <q-page-container>
      <router-view />
    </q-page-container>
  </q-layout>
</template>
