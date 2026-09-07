<script setup lang="ts">
import { ref } from 'vue'
import { createClient } from '@connectrpc/connect'
import { transport } from './lib/transport'
import { SystemService } from './gen/nostra/system/v1/system_pb'
import type { PingResponse } from './gen/nostra/system/v1/system_pb'

const client = createClient(SystemService, transport)

const message = ref('hallo')
const response = ref<PingResponse | null>(null)
const error = ref<string | null>(null)
const pending = ref(false)

async function ping() {
  pending.value = true
  error.value = null

  try {
    response.value = await client.ping({ message: message.value })
  } catch (err) {
    error.value = err instanceof Error ? err.message : String(err)
    response.value = null
  } finally {
    pending.value = false
  }
}

function formatTimestamp(value?: { seconds: bigint; nanos: number }) {
  if (!value) {
    return '–'
  }

  return new Date(Number(value.seconds) * 1000 + value.nanos / 1e6).toISOString()
}
</script>

<template>
  <UApp>
    <div class="mx-auto flex min-h-screen max-w-md flex-col justify-center gap-4 p-4">
      <UCard>
        <template #header>
          <h1 class="text-lg font-semibold">nostra</h1>
        </template>

        <div class="flex gap-2">
          <UInput v-model="message" class="flex-1" placeholder="Nachricht" />
          <UButton :loading="pending" @click="ping">Ping</UButton>
        </div>

        <UAlert
          v-if="error"
          class="mt-4"
          color="error"
          variant="subtle"
          title="Fehler"
          :description="error"
        />

        <dl v-if="response" class="mt-4 grid grid-cols-[auto_1fr] gap-x-4 gap-y-1 text-sm">
          <dt class="text-muted">Echo</dt>
          <dd>{{ response.echo }}</dd>
          <dt class="text-muted">Server</dt>
          <dd class="tabular-nums">{{ formatTimestamp(response.serverTime) }}</dd>
          <dt class="text-muted">Datenbank</dt>
          <dd class="tabular-nums">{{ formatTimestamp(response.dbTime) }}</dd>
          <dt class="text-muted">Pings</dt>
          <dd class="tabular-nums">{{ response.pingCount }}</dd>
          <dt class="text-muted">Version</dt>
          <dd>{{ response.version }}</dd>
        </dl>
      </UCard>
    </div>
  </UApp>
</template>
