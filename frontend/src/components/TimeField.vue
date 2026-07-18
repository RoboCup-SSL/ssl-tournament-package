<script setup lang="ts">
// A 24-hour time field. Mobile keeps the native OS picker (a good clock);
// desktop uses a clean q-time clock popup instead of Chromium's 3-column,
// AM/PM, infinite-scroll control. Model is a naive "HH:MM" string. On desktop
// the popup auto-closes once the minute is chosen (hour = 1st pick, minute = 2nd).
import { ref } from 'vue'
import { useQuasar } from 'quasar'
import type { QPopupProxy } from 'quasar'

defineProps<{ modelValue: string; label?: string }>()
const emit = defineEmits<{ 'update:modelValue': [string] }>()
const $q = useQuasar()

const proxy = ref<QPopupProxy>()
const picks = ref(0)

function set(value: string | number | null) {
  emit('update:modelValue', value == null ? '' : String(value))
}
function onPick(value: string | number | null) {
  set(value)
  picks.value += 1
  if (picks.value >= 2) proxy.value?.hide()
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
          @before-show="picks = 0"
        >
          <q-time :model-value="modelValue" format24h @update:model-value="onPick">
            <div class="row items-center justify-end">
              <q-btn v-close-popup label="Close" color="primary" flat />
            </div>
          </q-time>
        </q-popup-proxy>
      </q-icon>
    </template>
  </q-input>
</template>
