<script setup lang="ts">
// Fields section: list, add, rename, and delete a tournament's fields.
import { ref, watch } from 'vue'
import { useRoute } from 'vue-router'
import { useQuasar } from 'quasar'
import { useFieldsStore } from '@/store/fields'
import type { Field } from '@/api/types'

const route = useRoute()
const $q = useQuasar()
const store = useFieldsStore()

watch(
  () => route.params.id,
  (id) => { if (typeof id === 'string') void store.fetch(Number(id)) },
  { immediate: true },
)

const dialog = ref(false)
const editing = ref<Field | null>(null)
const name = ref('')
const busy = ref(false)

function openAdd() {
  editing.value = null
  name.value = ''
  dialog.value = true
}

function openEdit(field: Field) {
  editing.value = field
  name.value = field.name
  dialog.value = true
}

async function submit() {
  if (!name.value.trim()) return
  busy.value = true
  try {
    if (editing.value) await store.update(editing.value.id, { name: name.value.trim() })
    else await store.create({ name: name.value.trim() })
    if (store.error) {
      $q.notify({ type: 'negative', message: store.error })
    } else {
      dialog.value = false
      $q.notify({ type: 'positive', message: 'Saved' })
    }
  } finally {
    busy.value = false
  }
}

function confirmDelete(field: Field) {
  $q.dialog({ title: 'Delete field', message: `Delete "${field.name || 'this field'}"?`, cancel: true })
    .onOk(async () => {
      await store.remove(field.id)
      if (store.error) $q.notify({ type: 'negative', message: store.error })
      else $q.notify({ type: 'positive', message: 'Deleted' })
    })
}
</script>

<template>
  <q-page padding>
    <div class="row items-center q-mb-md">
      <div class="text-h6">Fields</div>
      <q-space />
      <q-btn color="primary" icon="add" label="Add field" @click="openAdd" />
    </div>

    <q-banner v-if="store.error && !store.items.length" class="bg-negative text-white q-mb-md">
      {{ store.error }}
    </q-banner>
    <q-spinner v-if="store.loading && !store.items.length" size="2em" />

    <q-list v-else-if="store.items.length" bordered separator>
      <q-item v-for="f in store.items" :key="f.id">
        <q-item-section>{{ f.name || '(unnamed)' }}</q-item-section>
        <q-item-section side>
          <div class="row items-center no-wrap">
            <q-btn flat round dense icon="edit" @click="openEdit(f)" />
            <q-btn flat round dense icon="delete" color="negative" @click="confirmDelete(f)" />
          </div>
        </q-item-section>
      </q-item>
    </q-list>
    <div v-else class="text-grey">No fields yet.</div>

    <q-dialog v-model="dialog">
      <q-card style="min-width: 320px">
        <q-card-section class="text-h6">{{ editing ? 'Edit field' : 'New field' }}</q-card-section>
        <q-card-section class="q-pt-none">
          <q-input v-model="name" label="Name" autofocus @keyup.enter="submit" />
        </q-card-section>
        <q-card-actions align="right">
          <q-btn v-close-popup flat label="Cancel" />
          <q-btn color="primary" :label="editing ? 'Save' : 'Create'" :loading="busy" @click="submit" />
        </q-card-actions>
      </q-card>
    </q-dialog>
  </q-page>
</template>
