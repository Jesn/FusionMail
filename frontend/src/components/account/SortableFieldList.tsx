import {
  DndContext,
  closestCenter,
  KeyboardSensor,
  PointerSensor,
  useSensor,
  useSensors,
  type DragEndEvent,
} from '@dnd-kit/core';
import {
  SortableContext,
  sortableKeyboardCoordinates,
  useSortable,
  verticalListSortingStrategy,
} from '@dnd-kit/sortable';
import { CSS } from '@dnd-kit/utilities';
import { GripVertical } from 'lucide-react';
import { FIELD_LABELS, type FieldType } from './import-types';

interface SortableFieldListProps {
  fields: FieldType[];
  onChange: (fields: FieldType[]) => void;
}

const SortableItem = ({ field, index }: { field: FieldType; index: number }) => {
  const { attributes, listeners, setNodeRef, transform, transition, isDragging } = useSortable({
    id: field,
  });

  const style = {
    transform: CSS.Transform.toString(transform),
    transition,
    zIndex: isDragging ? 10 : undefined,
  };

  return (
    <div
      ref={setNodeRef}
      style={style}
      className={`flex items-center gap-2 rounded-md border bg-background px-3 py-2 ${
        isDragging ? 'border-primary shadow-md opacity-90' : 'border-border'
      }`}
    >
      <button
        type="button"
        className="cursor-grab active:cursor-grabbing text-muted-foreground hover:text-foreground touch-none"
        {...attributes}
        {...listeners}
      >
        <GripVertical className="h-4 w-4" />
      </button>
      <span className="flex h-5 w-5 items-center justify-center rounded bg-muted text-xs font-medium text-muted-foreground">
        {index + 1}
      </span>
      <span className="text-sm font-medium">{FIELD_LABELS[field]}</span>
      <code className="ml-auto text-xs text-muted-foreground">{field}</code>
    </div>
  );
};

export const SortableFieldList = ({ fields, onChange }: SortableFieldListProps) => {
  const sensors = useSensors(
    useSensor(PointerSensor, { activationConstraint: { distance: 5 } }),
    useSensor(KeyboardSensor, { coordinateGetter: sortableKeyboardCoordinates }),
  );

  const handleDragEnd = (event: DragEndEvent) => {
    const { active, over } = event;
    if (!over || active.id === over.id) return;

    const oldIndex = fields.indexOf(active.id as FieldType);
    const newIndex = fields.indexOf(over.id as FieldType);
    if (oldIndex === -1 || newIndex === -1) return;

    const next = [...fields];
    const [moved] = next.splice(oldIndex, 1);
    next.splice(newIndex, 0, moved);
    onChange(next);
  };

  return (
    <DndContext sensors={sensors} collisionDetection={closestCenter} onDragEnd={handleDragEnd}>
      <SortableContext items={fields} strategy={verticalListSortingStrategy}>
        <div className="space-y-1.5">
          {fields.map((field, index) => (
            <SortableItem key={field} field={field} index={index} />
          ))}
        </div>
      </SortableContext>
    </DndContext>
  );
};