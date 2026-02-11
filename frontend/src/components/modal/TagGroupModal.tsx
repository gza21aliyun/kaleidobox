import { models } from "../../../wailsjs/go/models";
import { useEffect, useState } from "react";
import { ListTags, ListGroups, UpdateTagsGroup, DeleteTagGroup } from "../../../wailsjs/go/service/TagService";

import { toast } from "react-hot-toast";

// 标签分组弹窗组件
interface GroupModalProps {
  isOpen: boolean;
  onClose: () => void;
  mode: 'create' | 'edit';
  groupName?: string;
  allTags: models.Tag[];
  existingGroups: string[];
  onSave: (groupName: string, selectedTags: string[]) => Promise<void>;
}

export function TagGroupModal({ isOpen, onClose, mode, groupName, allTags, existingGroups, onSave }: GroupModalProps) {
  const [newGroupName, setNewGroupName] = useState(groupName || '');
  const [selectedTags, setSelectedTags] = useState<string[]>([]);
  const [isSaving, setIsSaving] = useState(false);

  // 编辑模式下初始化选中的标签
  useEffect(() => {
    if (mode === 'edit' && groupName) {
      const tagsInGroup = allTags.filter(tag => tag.group === groupName).map(tag => tag.name);
      setSelectedTags(tagsInGroup);
    } else {
      setSelectedTags([]);
    }
    setNewGroupName(groupName || '');
  }, [mode, groupName, allTags]);

  if (!isOpen) return null;

  const handleSave = async () => {
    if (!newGroupName.trim()) {
      toast.error('请输入分组名称');
      return;
    }

    if (mode === 'create' && existingGroups.includes(newGroupName)) {
      toast.error('分组名称已存在');
      return;
    }

    if (selectedTags.length === 0) {
      toast.error('请选择至少一个标签');
      return;
    }

    try {
      setIsSaving(true);
      await onSave(newGroupName, selectedTags);
      toast.success(mode === 'create' ? '分组创建成功' : '分组更新成功');
      onClose();
    } catch (error) {
      console.error('保存分组失败:', error);
      toast.error('保存分组失败');
    } finally {
      setIsSaving(false);
    }
  };

  const handleTagToggle = (tagName: string) => {
    setSelectedTags(prev => 
      prev.includes(tagName) 
        ? prev.filter(name => name !== tagName)
        : [...prev, tagName]
    );
  };

  return (
    <div className="fixed inset-0 bg-black/50 flex items-center justify-center z-50 p-4">
      <div className="bg-white dark:bg-brand-800 rounded-lg max-w-2xl w-full max-h-[80vh] overflow-hidden">
        <div className="p-6 border-b border-brand-200 dark:border-brand-700">
          <h2 className="text-xl font-bold text-brand-900 dark:text-white">
            {mode === 'create' ? '创建标签分组' : '编辑标签分组'}
          </h2>
        </div>
        
        <div className="p-6 overflow-y-auto max-h-[60vh]">
          <div className="mb-4">
            <label className="block text-sm font-medium text-brand-700 dark:text-brand-300 mb-2">
              分组名称
            </label>
            <input
              type="text"
              value={newGroupName}
              onChange={(e) => setNewGroupName(e.target.value)}
              className="w-full px-3 py-2 border border-brand-200 dark:border-brand-700 rounded-lg bg-white dark:bg-brand-700 text-brand-900 dark:text-white focus:outline-none focus:ring-2 focus:ring-brand-500"
              placeholder="输入分组名称"
            />
          </div>
          
          <div className="mb-4">
            <label className="block text-sm font-medium text-brand-700 dark:text-brand-300 mb-2">
              选择标签 ({selectedTags.length} 个已选择)
            </label>
            <div className="grid grid-cols-2 sm:grid-cols-3 gap-2 max-h-60 overflow-y-auto p-2 border border-brand-200 dark:border-brand-700 rounded-lg bg-brand-50 dark:bg-brand-900/20">
              {allTags.map(tag => (
                <div
                  key={tag.name}
                  onClick={() => handleTagToggle(tag.name)}
                  className={`
                    p-2 rounded cursor-pointer transition-colors flex items-center
                    ${selectedTags.includes(tag.name)
                      ? 'bg-brand-500 text-white'
                      : 'bg-white dark:bg-brand-800 text-brand-700 dark:text-brand-300 hover:bg-brand-100 dark:hover:bg-brand-700'
                    }
                  `}
                >
                  <div className={`
                    w-4 h-4 rounded border mr-2 flex items-center justify-center
                    ${selectedTags.includes(tag.name)
                      ? 'bg-white border-white'
                      : 'border-brand-300 dark:border-brand-600'
                    }
                  `}>
                    {selectedTags.includes(tag.name) && (
                      <div className="i-mdi-check text-xs text-brand-500"></div>
                    )}
                  </div>
                  <span className="truncate text-sm">{tag.name}</span>
                </div>
              ))}
            </div>
          </div>
        </div>
        
        <div className="p-6 border-t border-brand-200 dark:border-brand-700 flex justify-end gap-3">
          <button
            onClick={onClose}
            className="px-4 py-2 text-brand-700 dark:text-brand-300 hover:bg-brand-100 dark:hover:bg-brand-700 rounded-lg transition-colors"
            disabled={isSaving}
          >
            取消
          </button>
          <button
            onClick={handleSave}
            disabled={isSaving}
            className="px-4 py-2 bg-brand-500 text-white rounded-lg hover:bg-brand-600 disabled:opacity-50 disabled:cursor-not-allowed transition-colors flex items-center"
          >
            {isSaving ? (
              <>
                <div className="i-mdi-loading animate-spin mr-2"></div>
                保存中...
              </>
            ) : '保存'}
          </button>
        </div>
      </div>
    </div>
  );
}
