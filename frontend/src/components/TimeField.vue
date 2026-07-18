<script setup lang="ts">
// A 24-hour time field. Mobile keeps the native OS picker (a good clock);
// desktop uses a clean q-time clock popup instead of Chromium's 3-column,
// AM/PM, infinite-scroll control. Model is a naive "HH:MM" string.
import { useQuasar } from 'quasar'

defineProps<{ modelValue: string; label?: string }>()
const emit = defineEmits<{ 'update:modelValue': [string] }>()
const $q = useQuasar()

function set(value: string | number | null) {
  emit('update:modelValue', value == null ? '' : String(value))
}
</script>

<template>
  <q-input
    v-if="$q.platform.is.mobile"
    :model-value="modelValue"
    :label="label"
    type="time"
    stack-label
    clearable
    @update:model-value="set"
  />
  <q-input
    v-else
    :model-value="modelValue"
    :label="label"
    stack-label
    readonly
    clearable
    placeholder="--:--"
    @clear="set('')"
  >
    <template #append>
      <q-icon name="schedule" class="cursor-pointer">
        <q-popup-proxy cover transition-show="scale" transition-hide="scale">
          <q-time :model-value="modelValue" format24h @update:model-value="set">
            <div class="row items-center justify-end">
              <q-btn v-close-popup label="Close" color="primary" flat />
            </div>
          </q-time>
        </q-popup-proxy>
      </q-icon>
    </template>
  </q-input>
</template>
