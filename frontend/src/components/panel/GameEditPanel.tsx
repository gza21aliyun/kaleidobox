import type { models } from "../../../wailsjs/go/models";
import { toast } from "react-hot-toast";
import { OpenLocalPath, SearchSave, SelectFile } from "../../../wailsjs/go/service/GameService";
import { BetterButton } from "../ui/BetterButton";
import { useEffect, useRef, useState } from "react";
import { useTranslation } from "react-i18next";
import { OpenFolder } from "../../../wailsjs/go/service/BackupService";
import { BetterSelect } from "../ui/BetterSelect";
import { BetterSwitch } from "../ui/BetterSwitch";
import { getFolderPath } from "../utils/Utility";
import { BatchUpdateModal } from "../../components/modal/BatchUpdateModal";
import { GetAllVMs, OpenGamePathInsideVm } from "../../../wailsjs/go/service/VMService";
import { v } from "@unocss/preset-wind3/dist/rules-Dd5IWQsx.mjs";

interface GameEditFormProps {
  game: models.Game;
  onGameChange: (game: models.Game) => void;
  onDelete: () => void;
  onSelectExecutable: () => void;
  onSelectSaveDirectory: () => void;
  onSelectSaveFile: () => void;
  onSelectCoverImage: () => void;
  onUpdateFromRemote?: () => void;
  onLoadGame?: () => void;
}

export function GameEditPanel({
  game,
  onGameChange,
  onDelete,
  onSelectExecutable,
  onSelectSaveDirectory,
  onSelectSaveFile,
  onSelectCoverImage,
  onUpdateFromRemote,
  onLoadGame,
}: GameEditFormProps) {
  const { t } = useTranslation();
  const [isBatchUpdateOpen, setIsBatchUpdateOpen] = useState(false);
  const [vms, setVms] = useState<models.Vms[]>([]);

  useEffect(() => {
    // 获取虚拟机列表
    GetAllVMs().then(setVms).catch(console.error);
  }, []);

  const onSearchSaveDirectory = () => {
    // if ()
    SearchSave(game).then((newGame) => {
      onGameChange(newGame);
      toast.success("成功搜索到保存目录！");
    }).catch((error) => {
      console.error("Failed to search save directory:", error);
      toast.error("搜索保存目录失败"+error);
    });
  }

  const vmOptions = [
              { value: "", label: t('gameEdit.noVm') },
              ...vms.map(vm => ({
                value: vm.vm_id!,
                label: `${vm.vm_name} (${vm.vm_type})`
              }))
            ]
  console.log("vms options", vmOptions)


  return (
    <div className="glass-panel mx-auto bg-white dark:bg-brand-800 p-8 rounded-lg shadow-sm">
      <div className="space-y-6">
        <div>
          <label className="block text-sm font-medium text-brand-700 dark:text-brand-300 mb-1">
            {t('gameEdit.name')}
          </label>
          <input
            type="text"
            value={game.name}
            onChange={e => onGameChange({ ...game, name: e.target.value } as models.Game)}
            className="glass-input w-full px-3 py-2 border border-brand-300 dark:border-brand-600 rounded-md bg-white dark:bg-brand-700 text-brand-900 dark:text-white focus:ring-2 focus:ring-neutral-500 outline-none"
          />
        </div>

        <div>
          <label className="block text-sm font-medium text-brand-700 dark:text-brand-300 mb-1">
            {t('gameEdit.searchName')}
          </label>
          <input
            type="text"
            value={game.search_name}
            onChange={e => onGameChange({ ...game, search_name: e.target.value } as models.Game)}
            className="glass-input w-full px-3 py-2 border border-brand-300 dark:border-brand-600 rounded-md bg-white dark:bg-brand-700 text-brand-900 dark:text-white focus:ring-2 focus:ring-neutral-500 outline-none"
          />
        </div>

        <div>
          <label className="block text-sm font-medium text-brand-700 dark:text-brand-300 mb-1">
            {t('gameEdit.coverImage')}
          </label>
          <div className="flex gap-2">
            <input
              type="text"
              value={game.cover_url}
              onChange={e => onGameChange({ ...game, cover_url: e.target.value } as models.Game)}
              placeholder={t('gameEdit.coverImagePlaceholder')}
              className="glass-input flex-1 px-3 py-2 border border-brand-300 dark:border-brand-600 rounded-md bg-white dark:bg-brand-700 text-brand-900 dark:text-white focus:ring-2 focus:ring-neutral-500 outline-none"
            />
            <BetterButton onClick={onSelectCoverImage} icon="i-mdi-image" title={t('gameEdit.selectImage')} />
          </div>
          <p className="mt-1 text-xs text-brand-500">{t('gameEdit.coverImageHint')}</p>
        </div>

        <div>
          <label className="block text-sm font-medium text-brand-700 dark:text-brand-300 mb-1">
            {t('gameEdit.developer')}
          </label>
          <input
            type="text"
            value={game.company}
            onChange={e => onGameChange({ ...game, company: e.target.value } as models.Game)}
            className="glass-input w-full px-3 py-2 border border-brand-300 dark:border-brand-600 rounded-md bg-white dark:bg-brand-700 text-brand-900 dark:text-white focus:ring-2 focus:ring-neutral-500 outline-none"
          />
        </div>

        <div>
          <label className="block text-sm font-medium text-brand-700 dark:text-brand-300 mb-1">
            {t('gameEdit.gamePath')}
          </label>
          <div className="flex gap-2">
            <input
              type="text"
              value={game.path}
              onChange={e => onGameChange({ ...game, path: e.target.value } as models.Game)}
              className="glass-input flex-1 px-3 py-2 border border-brand-300 dark:border-brand-600 rounded-md bg-white dark:bg-brand-700 text-brand-900 dark:text-white focus:ring-2 focus:ring-neutral-500 outline-none"
            />
            <div className="flex items-center gap-1">
              <BetterButton onClick={onSelectExecutable} icon="i-mdi-file" title={t('gameEdit.selectFile')} />
              <BetterButton
                onClick={async () => {
                  try {
                    await OpenLocalPath(game.path);
                  }
                  catch {
                    toast.error(t('gameEdit.openPathError'));
                  }
                }}
                disabled={!game.path}
                icon="i-mdi-folder-open"
                title={t('gameEdit.openLocation')}
              />
            </div>
          </div>
          <label className="block text-sm font-medium text-brand-700 dark:text-brand-300 mb-1">
            {t('gameEdit.arguments')}
          </label>
          <div className="flex gap-2">
            <input
              type="text"
              value={game.arguments}
              onChange={e => onGameChange({ ...game, arguments: e.target.value.trim() } as models.Game)}
              className="glass-input flex-1 px-3 py-2 border border-brand-300 dark:border-brand-600 rounded-md bg-white dark:bg-brand-700 text-brand-900 dark:text-white focus:ring-2 focus:ring-neutral-500 outline-none"
            />
          </div>
        </div>

        <div>
          <label className="block text-sm font-medium text-brand-700 dark:text-brand-300 mb-1">
            {t('gameEdit.savePath')}
          </label>
          <div className="flex gap-2">
            <input
              type="text"
              value={game.save_path || ""}
              onChange={e => onGameChange({ ...game, save_path: e.target.value } as models.Game)}
              placeholder={t('gameEdit.savePathPlaceholder')}
              className="glass-input flex-1 px-3 py-2 border border-brand-300 dark:border-brand-600 rounded-md bg-white dark:bg-brand-700 text-brand-900 dark:text-white focus:ring-2 focus:ring-neutral-500 outline-none"
            />
            <button
              type="button"
              onClick={onSelectSaveDirectory}
              className="glass-btn-neutral px-4 py-2 bg-brand-100 dark:bg-brand-700 text-brand-700 dark:text-brand-300 rounded-md hover:bg-brand-200 dark:hover:bg-brand-600 transition-colors"
            >
              {t('gameEdit.select')}
            </button>

            <button
              type="button"
              onClick={onSearchSaveDirectory}
              className="glass-btn-neutral px-4 py-2 bg-brand-100 dark:bg-brand-700 text-brand-700 dark:text-brand-300 rounded-md hover:bg-brand-200 dark:hover:bg-brand-600 transition-colors"
            >
              搜索
            </button>
            {game.save_path && (
              <BetterButton
                onClick={async () => {
                  try {
                    await OpenLocalPath(game.save_path);
                  }
                  catch {
                    toast.error(t('gameEdit.openPathError'));
                  }
                }}
                disabled={!game.save_path}
                icon="i-mdi-folder-open"
                title={t('gameEdit.openLocation')}
              />
            )}
          </div>
          <p className="mt-1 text-xs text-brand-500">{t('gameEdit.savePathHint')}</p>
        </div>

        <div>
          <label className="block text-sm font-medium text-brand-700 dark:text-brand-300 mb-1">
            {t('gameEdit.pvPathLabel')}
          </label>
          <div className="flex gap-2">
            <input
              type="text"
              value={game.pv_path || ""}
              onChange={e => onGameChange({ ...game, pv_path: e.target.value } as models.Game)}
              placeholder={t('gameEdit.pvPathPlaceholder')}
              className="glass-input flex-1 px-3 py-2 border border-brand-300 dark:border-brand-600 rounded-md bg-white dark:bg-brand-700 text-brand-900 dark:text-white focus:ring-2 focus:ring-neutral-500 outline-none"
            />
            <button
              type="button"
              onClick={async () => {
                try {
                  const selection = await SelectFile("视频文件", "*.exe;*.bat;*.cmd;*.lnk");
                  if (selection) {
                    onGameChange({ ...game, pv_path: selection } as models.Game);
                  }
                } catch (error) {
                  toast.error(t('gameEdit.selectFileFailed'));
                }
              }}
              className="glass-btn-neutral px-4 py-2 bg-brand-100 dark:bg-brand-700 text-brand-700 dark:text-brand-300 rounded-md hover:bg-brand-200 dark:hover:bg-brand-600 transition-colors"
            >
              {t('gameEdit.select')}
            </button>
          </div>
          <p className="mt-1 text-xs text-brand-500">{t('gameEdit.pvPathHint')}</p>
        </div>

        <div>
          <label className="block text-sm font-medium text-brand-700 dark:text-brand-300 mb-1">
            {t('gameEdit.vmId')}
          </label>
           <div className="flex items-center gap-2">
            <BetterSelect
              value={game.vm_id!}
              onChange={value => onGameChange({ ...game, vm_id: value } as models.Game)}
              options={vmOptions}
              className="flex-1"
              // placeholder={t('common.pleaseSelect')}
            />
            <BetterButton
              onClick={async () => {
                if (!game.vm_id) {
                  toast.error(t('gameEdit.noVmSelected'));
                  return;
                }
                try {
                  await OpenGamePathInsideVm(game.id!);
                  toast.success(t('gameEdit.openVmPathSuccess'));
                } catch (error) {
                  toast.error(t('gameEdit.openVmPathError'));
                }
              }}
              disabled={!game.vm_id || !game.path}
              icon="i-mdi-folder-open"
              title={t('gameEdit.openInVm')}
            />
          </div>
          <span className="ml-2 text-sm text-brand-600 dark:text-brand-400">{t('gameEdit.vmIdHint')}</span>
        </div>

        <div>
          <label className="block text-sm font-medium text-brand-700 dark:text-brand-300 mb-1">
            {t('gameEdit.summary')}
          </label>
          <textarea
            value={game.summary}
            onChange={e => onGameChange({ ...game, summary: e.target.value } as models.Game)}
            rows={6}
            className="glass-input w-full px-3 py-2 border border-brand-300 dark:border-brand-600 rounded-md bg-white dark:bg-brand-700 text-brand-900 dark:text-white focus:ring-2 focus:ring-neutral-500 outline-none resize-none"
          />
        </div>

        <div className="grid grid-cols-2 gap-4">
          <div>
            <label className="block text-sm font-medium text-brand-700 dark:text-brand-300 mb-1">
              {t('gameEdit.sourceType')}
            </label>
            <BetterSelect
              value={game.source_type || ""}
              onChange={value => onGameChange({ ...game, source_type: value } as models.Game)}
              options={[
                { value: "", label: t('gameEdit.none') },
                { value: "local", label: t('sourceType.local') },
                { value: "bangumi", label: t('sourceType.bangumi') },
                { value: "vndb", label: t('sourceType.vndb') },
                { value: "ymgal", label: t('sourceType.ymgal') },
                { value: "dmm", label: t('sourceType.dmm') },
                { value: "eroscape", label: t('sourceType.eroscape') },
                { value: "dlsite", label: t('sourceType.dlsite') },
                { value: "getchu", label: t('sourceType.getchu') },
              ]}
            />
          </div>
          <div>
            <label className="block text-sm font-medium text-brand-700 dark:text-brand-300 mb-1">
              {t('gameEdit.sourceId')}
            </label>
            <input
              type="text"
              value={game.source_id || ""}
              onChange={e => onGameChange({ ...game, source_id: e.target.value } as models.Game)}
              placeholder={t('gameEdit.sourceIdPlaceholder')}
              className="glass-input w-full px-3 py-2 border border-brand-300 dark:border-brand-600 rounded-md bg-white dark:bg-brand-700 text-brand-900 dark:text-white focus:ring-2 focus:ring-neutral-500 outline-none"
            />
          </div>
        </div>

        <div className="flex justify-between pt-4">
          <div className="flex gap-4 justify-end w-full">
            {onUpdateFromRemote && (
              <BetterButton
                variant="primary"
                onClick={onUpdateFromRemote}
                icon="i-mdi-cloud-sync"
              >
                {t('gameEdit.updateFromRemote')}
              </BetterButton>
            )}
            {onLoadGame && (
              <button
                type="button"
                onClick={() => setIsBatchUpdateOpen(true)}
                className="glass-btn-neutral px-6 py-2 bg-accent-500 text-white rounded-md hover:bg-accent-700 transition-colors focus:ring-2 focus:ring-offset-2 focus:ring-accent-500"
              >
                {t('gameEdit.updateFromSource')}
              </button>
            )}
            <BetterButton
              variant="danger"
              onClick={onDelete}
              icon="i-mdi-trash-can-outline"
            >
              {t('gameEdit.delete')}
            </BetterButton>
          </div>
        </div>
        
      <BatchUpdateModal
        isOpen={isBatchUpdateOpen}
        onClose={() => setIsBatchUpdateOpen(false)}
        onUpdateComplete={() => {onLoadGame?.();}}
        games={[game]}
      />
      </div>
    </div>
  );
}
