import { models } from "../../../wailsjs/go/models";
import { useTranslation } from 'react-i18next';
import { useDrag, useDrop } from 'react-dnd';

// 拖拽类型定义
interface DragItem {
  type: 'tag';
  tagName: string;
}

// 可拖拽的标签组件
export function DraggableTag({ tag }: { tag: models.Tag }) {
  const { t } = useTranslation();
  const [{ isDragging }, drag] = useDrag(() => ({
    type: 'tag',
    item: { type: 'tag', tagName: tag.name },
    collect: (monitor) => ({
      isDragging: !!monitor.isDragging(),
    }),
  }));

  return (
    <span
      ref={drag}
      className={`
        inline-flex items-center px-3 py-1 rounded-full text-sm font-medium cursor-move
        ${tag.is_h
          ? 'bg-red-100 text-red-800 dark:bg-red-900/30 dark:text-red-200 border border-red-200 dark:border-red-800' 
          : tag.is_spoiler
          ? 'bg-yellow-100 text-yellow-800 dark:bg-yellow-900/30 dark:text-yellow-200 border border-yellow-200 dark:border-yellow-800'
          : 'bg-brand-100 text-brand-800 dark:bg-brand-900/30 dark:text-brand-200 border border-brand-200 dark:border-brand-700'
        }
        ${tag.block_modify ? 'opacity-75' : ''}
        ${isDragging ? 'opacity-50 scale-95' : 'hover:scale-105'}
        transition-all duration-200
      `}
    >
      {tag.name}
      {tag.is_h && (
        <div className="ml-1 text-xs" title={t('tag.tags.adultContent')}>
          <div className="i-mdi-alert-circle-outline"></div>
        </div>
      )}
      {tag.is_spoiler && (
        <div className="ml-1 text-xs" title={t('tag.tags.spoiler')}>
          <div className="i-mdi-eye-off-outline"></div>
        </div>
      )}
      {tag.block_modify && (
        <div className="ml-1 text-xs" title={t('tag.tags.blockModify')}>
          <div className="i-mdi-lock-outline"></div>
        </div>
      )}
      <div className="ml-1 text-xs opacity-70">
        <div className="i-mdi-drag-horizontal"></div>
      </div>
    </span>
  );
}

// 可放置的分组组件
export function DroppableGroup({ 
  groupName, 
  tagCount, 
  tags, 
  onDrop,
  onEdit,
  onDelete,

}: { 
  groupName: string; 
  tagCount: number; 
  tags: models.Tag[];
  onDrop: (tagName: string, targetGroup: string) => void;
  onEdit: (groupName: string) => void;
  onDelete: (groupName: string) => void;
}) {
  const { t } = useTranslation();
  const [{ isOver, canDrop }, drop] = useDrop(() => ({
    accept: 'tag',
    drop: (item: DragItem) => onDrop(item.tagName, groupName),
    collect: (monitor) => ({
      isOver: !!monitor.isOver(),
      canDrop: !!monitor.canDrop(),
    }),
  }));

  

  return (
    <div 
      ref={drop}
      className={`
        bg-white dark:bg-brand-800/30 rounded-lg p-4 border transition-all duration-200
        ${isOver && canDrop 
          ? 'border-brand-500 bg-brand-50 dark:bg-brand-900/50 shadow-lg scale-[1.02]' 
          : 'border-brand-200 dark:border-brand-700 hover:shadow-md'
        }
      `}
    >
      <div className="flex items-center justify-between mb-3">
        <h3 className="font-semibold text-brand-900 dark:text-white flex items-center">
          <div className="i-mdi-folder mr-2 text-brand-600 dark:text-brand-400"></div>
          {groupName}
        </h3>
        <div className="flex gap-1">
          <button
            onClick={() => onEdit(groupName)}
            className="p-1 text-brand-600 hover:text-brand-800 dark:text-brand-400 dark:hover:text-brand-200"
            title={t('tag.actions.editGroup')}
          >
            <div className="i-mdi-pencil text-sm"></div>
          </button>
          <button
            onClick={() => onDelete(groupName)}
            className="p-1 text-red-600 hover:text-red-800"
            title={t('tag.actions.deleteGroup')}
          >
            <div className="i-mdi-delete text-sm"></div>
          </button>
        </div>
      </div>
      <div className="text-sm text-brand-600 dark:text-brand-400 mb-2">
        {t('tag.labels.containsCount', { count: tagCount })}
        {isOver && canDrop && (
          <span className="ml-2 text-brand-500 font-medium">{t('tag.hints.releaseToAdd')}</span>
        )}
      </div>
      <div className="flex flex-wrap gap-1">
        {tags?.map(tag => (
          <span
            key={tag.name}
            className="px-2 py-1 bg-brand-100 dark:bg-brand-900/30 text-brand-800 dark:text-brand-200 text-xs rounded-full"
          >
            {tag.name}
          </span>
        ))}
      </div>
    </div>
  );
}