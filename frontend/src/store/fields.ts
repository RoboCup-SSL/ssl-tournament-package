// Pinia store for a tournament's fields (the Fields section). The reusable
// list-CRUD pattern: fetch by tournament, then create/update/remove re-fetch.
import { defineStore } from 'pinia'
import { api } from '@/api/client'
import type { Field, FieldInput } from '@/api/types'

export const useFieldsStore = defineStore('fields', {
  state: () => ({
    items: [] as Field[],
    tournamentId: 0,
    loading: false,
    error: '',
  }),
  actions: {
    // fetch loads the tournament's fields and records the tournament id.
    // A silent fetch (polling) skips loading/error UI and only reassigns items
    // when they changed, so an unchanged poll triggers no re-render.
    async fetch(tournamentId: number, silent = false) {
      this.tournamentId = tournamentId
      if (!silent) {
        this.loading = true
        this.error = ''
      }
      try {
        const items = await api.get<Field[]>(`/api/fields?tournament_id=${tournamentId}`)
        if (JSON.stringify(items) !== JSON.stringify(this.items)) this.items = items
      } catch (failure) {
        if (!silent) this.error = failure instanceof Error ? failure.message : String(failure)
      } finally {
        if (!silent) this.loading = false
      }
    },
    // create adds a field to the current tournament, then refreshes.
    async create(input: FieldInput) {
      this.error = ''
      try {
        await api.post<Field>('/api/fields', { tournament_id: this.tournamentId, ...input })
        await this.fetch(this.tournamentId)
      } catch (failure) {
        this.error = failure instanceof Error ? failure.message : String(failure)
      }
    },
    async update(id: number, input: FieldInput) {
      this.error = ''
      try {
        await api.patch<Field>(`/api/fields/${id}`, input)
        await this.fetch(this.tournamentId)
      } catch (failure) {
        this.error = failure instanceof Error ? failure.message : String(failure)
      }
    },
    async remove(id: number) {
      this.error = ''
      try {
        await api.del(`/api/fields/${id}`)
        await this.fetch(this.tournamentId)
      } catch (failure) {
        this.error = failure instanceof Error ? failure.message : String(failure)
      }
    },
  },
})
