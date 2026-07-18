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
    async fetch(tournamentId: number) {
      this.tournamentId = tournamentId
      this.loading = true
      this.error = ''
      try {
        this.items = await api.get<Field[]>(`/api/fields?tournament_id=${tournamentId}`)
      } catch (failure) {
        this.error = failure instanceof Error ? failure.message : String(failure)
      } finally {
        this.loading = false
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
