import { useState } from 'react'
import type { EntityType } from '../../shared/api/types.ts'
import { Form, TextAreaField, text } from '../../shared/ui/Form.tsx'
import { useCreateComment } from './api.ts'

export function CommentForm({ entityType, entityCode }: { entityType: EntityType; entityCode: string }) {
  const create = useCreateComment(entityType, entityCode)
  // Al guardar, el formulario se vuelve a montar vacío.
  const [saved, setSaved] = useState(0)
  return (
    <Form key={saved} label="Comentar" submitLabel="Comentar" onSubmit={(data) => create.mutateAsync(text(data, 'body'))} onDone={() => setSaved(saved + 1)}>
      <TextAreaField name="body" label="Comentario" required />
    </Form>
  )
}
