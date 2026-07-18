<script setup lang="ts">
// A 24-hour time field. Mobile keeps the native OS picker (a good clock);
// desktop uses a clean q-time clock popup instead of Chromium's 3-column,
// AM/PM, infinite-scroll control. Model is a naive "HH:MM" string.
//
// The desktop popup auto-closes after the minute is chosen. We count *clicks on
// the clock face* (1st = hour, 2nd = minute), not value-change emits — q-time
// only emits when the value changes, so re-picking the same hour/minute would
// never advance an emit-based counter.
import { ref } from 'vue'
import { useQuasar } from 'quasar'
import type { QPopupProxy } from 'quasar'

defineProps<{ modelValue: string; label?: string }>()
const emit = defineEmits<{ 'update:modelValue': [string] }>()
const $q = useQuasar()

const proxy = ref<QPopupProxy>()
const clockClicks = ref(0)

function set(value: string | number | null) {
  emit('update:modelValue', value == null ? '' : String(value))
}
function onClockClick(event: MouseEvent) {
  if (!(event.target as HTMLElement).closest('[class*="q-time__clock"]')) return
  clockClicks.value += 1
  if (clockClicks.value >= 2) proxy.value?.hide()
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
        <q-popup-proxy
          ref="proxy"
          cover
          transition-show="scale"
          transition-hide="scale"
          @before-show="clockClicks = 0"
        >
          <div @click="onClockClick">
            <q-time :model-value="modelValue" format24h @update:model-value="set">
              <div class="row items-center justify-end">
                <q-btn v-close-popup label="Close" color="primary" flat />
              </div>
            </q-time>
          </div>
        </q-popup-proxy>
      </q-icon>
    </template>
  </q-input>
</template>
