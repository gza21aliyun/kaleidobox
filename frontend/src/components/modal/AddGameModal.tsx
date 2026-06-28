import { useState } from "react";
import { toast } from "react-hot-toast";
import { enums, models, vo } from "../../../wailsjs/go/models";
import { AddGame, FetchMetadata, FetchMetadataByName, SelectCoverImageWithTempID,
   SelectGameExecutable } from "../../../wailsjs/go/service/GameService";
import { useAppStore } from "../../store";
import { BetterSelect } from "../ui/BetterSelect";
import i18next from "../../i18n/i18n";
const t = i18next.t;

interface AddGameModalProps {
  isOpen: boolean;
  onClose: () => void;
  onGameAdded: () => void;
}

// step: 1=选择程序, 2=搜索结果, 3=按ID搜索, 4=手动填写
type StepType = 1 | 2 | 3 | 4;

export function AddGameModal({ isOpen, onClose, onGameAdded }: AddGameModalProps) {
  const [step, setStep] = useState<StepType>(1);
  const { config } = useAppStore();
  const [executablePath, setExecutablePath] = useState("");
  const [gameName, setGameName] = useState("");
  const [metadataResults, setMetadataResults] = useState<vo.GameMetadataFromWebVO[]>([]);
  const [isLoading, setIsLoading] = useState(false);
  const [manualId, setManualId] = useState("");
  const [manualSource, setManualSource] = useState<enums.SourceType>(enums.SourceType.BANGUMI);

  // 手动添加表单字段
  const [manualCoverUrl, setManualCoverUrl] = useState("");
  const [manualCompany, setManualCompany] = useState("");
  const [manualSummary, setManualSummary] = useState("");

  if (!isOpen)
    return null;

  const resetAndClose = () => {
    setStep(1);
    setExecutablePath("");
    setGameName("");
    setMetadataResults([]);
    setManualId("");
    setManualCoverUrl("");
    setManualCompany("");
    setManualSummary("");
    onClose();
  };

  const handleSelectExecutable = async () => {
    try {
      var path = "";
      path = await SelectGameExecutable(); 
      
      if (path) {
        setExecutablePath(path);
        // Extract parent folder name
        // Windows path usually uses backslash, but Wails might return forward slash or mixed.
        // Let's handle both.
        const normalizedPath = path.replace(/\\/g, "/");
        const parts = normalizedPath.split("/");
        // If it's a file, the last part is filename, the one before is parent folder
        if (parts.length > 1) {
          setGameName(parts[parts.length - 2]);
        }
      }
    }
    catch (error) {
      console.error("Failed to select executable:", error);
      toast.error(t('game.toasts.openSystemSelectorFailed'));
    }
  };

  const handleSearchByName = async () => {
    if (!gameName)
      return;
    setIsLoading(true);
    try {
      const results = await FetchMetadataByName(gameName);
      setMetadataResults(results || []);
      setStep(2);
    }
    catch (error) {
      console.error("Failed to fetch metadata:", error);
      toast.error(t('game.toasts.fetchMetadataFailed'));
    }
    finally {
      setIsLoading(false);
    }
  };

  const saveGame = async (game: models.Game) => {
    try {
      game.path = executablePath;
      // Ensure other fields are set if missing?
      // The backend AddGame handles ID generation if empty.
      await AddGame(game);
      onGameAdded();
      resetAndClose();
    }
    catch (error) {
      console.error("Failed to save game:", error);
      toast.error(t('game.toasts.saveGameFailed'));
    }
  };

  const handleSearchById = async () => {
    if (!manualId)
      return;
    setIsLoading(true);
    try {
      const request = new vo.MetadataRequest({
        source: manualSource,
        id: manualId,
      });
      const game = await FetchMetadata(request);
      if (game) {
        await saveGame(game);
      }
    }
    catch (error) {
      console.error("Failed to fetch metadata by ID:", error);
      toast.error(t('game.toasts.fetchMetadataByIdFailed'));
    }
    finally {
      setIsLoading(false);
    }
  };

  const handleSelectCoverImage = async () => {
    try {
      const coverUrl = await SelectCoverImageWithTempID();
      if (coverUrl) {
        setManualCoverUrl(coverUrl);
      }
    }
    catch (error) {
      console.error("Failed to select cover image:", error);
      toast.error(t('game.toasts.selectCoverImageFailed'));
    }
  };

  const handleManualSave = async () => {
    if (!gameName) {
      toast.error(t('game.toasts.pleaseFillGameName'));
      return;
    }
    setIsLoading(true);
    try {
      const game = new models.Game({
        name: gameName,
        path: executablePath,
        cover_url: manualCoverUrl,
        company: manualCompany,
        summary: manualSummary,
        source_type: enums.SourceType.LOCAL,
        status: enums.GameStatus.NOT_STARTED,
      });
      await AddGame(game);
      onGameAdded();
      resetAndClose();
    }
    catch (error) {
      console.error("Failed to save game manually:", error);
      toast.error("保存游戏失败");
    }
    finally {
      setIsLoading(false);
    }
  };

  return (
    <div className="fixed inset-0 z-50 flex items-center justify-center bg-black/50 backdrop-blur-sm">
      <div className="w-full max-w-2xl rounded-xl bg-white p-6 shadow-2xl dark:bg-brand-800">
        <div className="flex items-center justify-between">
          <h2 className="text-4xl font-bold text-brand-900 dark:text-white mb-6">{t('game.modals.addGame.title')}</h2>
          <button
            onClick={resetAndClose}
            className="i-mdi-close text-2xl text-brand-500 p-1 rounded-lg mb-6
              hover:bg-brand-100 hover:text-brand-700 focus:outline-none
              dark:text-brand-400 dark:hover:bg-brand-700 dark:hover:text-brand-200"
            aria-label={t('common.close')}
          />
        </div>

        {step === 1 && (
          <div className="space-y-6">
            <button
              onClick={handleSelectExecutable}
              className="flex w-full items-center justify-center rounded-lg bg-neutral-500 py-4 text-white transition hover:bg-neutral-600"
            >
              <div className="i-mdi-file-find mr-2 text-xl" />
              {t('game.modals.addGame.selectExecutable')}
            </button>

            <div>
              <input
                type="text"
                value={executablePath}
                readOnly
                placeholder={t('game.modals.addGame.selectExecutablePlaceholder')}
                className="box-border block w-full rounded-lg border border-brand-300 bg-brand-50 p-3 text-brand-900 dark:border-brand-600 dark:bg-brand-700 dark:text-white"
              />
            </div>

            <div>
              <label className="mb-2 block text-sm font-medium text-brand-900 dark:text-white">{t('game.modals.addGame.gameNameLabel')}</label>
              <input
                type="text"
                value={gameName}
                onChange={e => setGameName(e.target.value)}
                className="box-border block w-full rounded-lg border border-brand-300 bg-brand-50 p-3 text-brand-900 dark:border-brand-600 dark:bg-brand-700 dark:text-white"
              />
            </div>

            <div className="flex justify-end space-x-4">
              <button
                onClick={() => setStep(4)}
                disabled={!executablePath || !gameName}
                className="rounded-lg border border-brand-300 px-5 py-2.5 text-sm font-medium text-brand-700 hover:bg-brand-100 disabled:opacity-50 dark:border-brand-600 dark:text-brand-300 dark:hover:bg-brand-700"
              >
                {t('game.modals.addGame.manualAdd')}
              </button>
              {(config?.hide_old_matching_btns == false) && (
                <button
                  onClick={handleSearchByName}
                  disabled={!executablePath || !gameName || isLoading}
                  className="rounded-lg bg-neutral-600 px-5 py-2.5 text-sm font-medium text-white hover:bg-neutral-700 disabled:opacity-50"
                >
                  {isLoading ? t('game.modals.addGame.searching') : t('game.modals.addGame.searchMetadata')}
                </button>
              )}
              
            </div>
          </div>
        )}

        {step === 2 && (
          <div className="space-y-6">
            <p className="text-brand-600 dark:text-brand-300">{t('game.modals.addGame.whichResultExpected')}</p>

            <div className="flex max-h-[400px] flex-wrap justify-center gap-4 overflow-y-auto p-1">
              {metadataResults.filter(item => item.Game)
                .map((item, index) => (
                  <div
                    key={index}
                    onClick={() => saveGame(item.Game!)} // 使用非空断言，因为上面已过滤
                    className="w-44 cursor-pointer rounded-lg border border-brand-200 p-3 transition hover:border-neutral-500 hover:shadow-md dark:border-brand-700 dark:hover:border-neutral-400"
                  >
                    <div className="aspect-[3/4] w-full overflow-hidden rounded-md bg-brand-200 dark:bg-brand-700">
                      {item.Game!.cover_url
                        ? (
                            <img src={item.Game!.cover_url} alt={item.Game!.name} className="h-full w-full object-cover" referrerPolicy="no-referrer" draggable="false" onDragStart={e => e.preventDefault()} />
                          )
                        : (
                            <div className="flex h-full items-center justify-center text-brand-400">
                              <div className="i-mdi-image-off text-4xl" />
                            </div>
                          )}
                    </div>
                    <h3 className="mt-2 truncate text-sm font-bold text-brand-900 dark:text-white" title={item.Game!.name}>{item.Game!.name}</h3>
                    <p className="text-xs text-brand-500 dark:text-brand-400">
                      {t('game.modals.addGame.from')}
                      {" "}
                      {item.Source}
                    </p>
                  </div>
                ))}
            </div>

            <div className="flex items-center justify-between border-t border-brand-200 pt-4 dark:border-brand-700">
              <button
                onClick={() => setStep(1)}
                className="text-sm text-brand-500 hover:text-brand-700 dark:text-brand-400 dark:hover:text-brand-200"
              >
                &larr; {t('game.modals.addGame.backToPreviousStep')}
              </button>
              <div className="flex space-x-4">
                <div className="text-sm text-brand-500 dark:text-brand-400">
                  {t('game.modals.addGame.noneOfThem')}
                </div>
                <button
                  onClick={() => setStep(4)}
                  className="text-sm text-brand-500 hover:text-brand-700 dark:text-brand-400 dark:hover:text-brand-200"
                >
                  {t('game.modals.addGame.manualFill')}
                </button>
                <button
                  onClick={() => setStep(3)}
                  className="text-sm text-neutral-600 hover:text-neutral-800 dark:text-neutral-400 dark:hover:text-neutral-300"
                >
                  {t('game.modals.addGame.inputIdSearch')}
                </button>
              </div>
            </div>
          </div>
        )}

        {step === 3 && (
          <div className="space-y-6">
            <div>
              <label className="mb-2 block text-sm font-medium text-brand-900 dark:text-white">{t('game.modals.addGame.dataSource')}</label>
              <BetterSelect
                value={manualSource}
                onChange={value => setManualSource(value as enums.SourceType)}
                options={[
                  { value: enums.SourceType.BANGUMI, label: t('sourceType.bangumi') },
                  { value: enums.SourceType.VNDB, label: t('sourceType.vndb') },
                  { value: enums.SourceType.YMGAL, label: t('sourceType.ymgal') },
                ]}
              />
            </div>

            <div>
              <label className="mb-2 block text-sm font-medium text-brand-900 dark:text-white">{t('game.modals.addGame.gameId')}</label>
              <input
                type="text"
                value={manualId}
                onChange={e => setManualId(e.target.value)}
                placeholder={t('game.modals.addGame.gameIdPlaceholder')}
                className="box-border block w-full rounded-lg border border-brand-300 bg-brand-50 p-3 text-brand-900 dark:border-brand-600 dark:bg-brand-700 dark:text-white"
              />
            </div>

            <div className="flex justify-end space-x-4">
              <button
                onClick={() => setStep(2)}
                className="rounded-lg border border-brand-300 px-5 py-2.5 text-sm font-medium text-brand-700 hover:bg-brand-100 dark:border-brand-600 dark:text-brand-300 dark:hover:bg-brand-700"
              >
                {t('game.modals.addGame.back')}
              </button>
              <button
                onClick={handleSearchById}
                disabled={!manualId || isLoading}
                className="rounded-lg bg-neutral-600 px-5 py-2.5 text-sm font-medium text-white hover:bg-neutral-700 disabled:opacity-50"
              >
                {isLoading ? t('game.modals.addGame.searching') : t('game.modals.addGame.confirm')}
              </button>
            </div>
          </div>
        )}

        {step === 4 && (
          <div className="space-y-4">
            <p className="text-brand-600 dark:text-brand-300">{t('game.modals.addGame.manualFillInfo')}</p>

            <div>
              <label className="mb-2 block text-sm font-medium text-brand-900 dark:text-white">{t('game.modals.addGame.gameNameLabel')}</label>
              <input
                type="text"
                value={gameName}
                onChange={e => setGameName(e.target.value)}
                className="box-border block w-full rounded-lg border border-brand-300 bg-brand-50 p-3 text-brand-900 dark:border-brand-600 dark:bg-brand-700 dark:text-white"
              />
            </div>

            <div>
              <label className="mb-2 block text-sm font-medium text-brand-900 dark:text-white">{t('game.modals.addGame.coverImage')}</label>
              <div className="flex gap-2">
                <input
                  type="text"
                  value={manualCoverUrl}
                  onChange={e => setManualCoverUrl(e.target.value)}
                  placeholder={t('game.modals.addGame.coverImagePlaceholder')}
                  className="box-border block flex-1 rounded-lg border border-brand-300 bg-brand-50 p-3 text-brand-900 dark:border-brand-600 dark:bg-brand-700 dark:text-white"
                />
                <button
                  type="button"
                  onClick={handleSelectCoverImage}
                  className="rounded-lg bg-brand-100 px-4 py-2 text-brand-700 hover:bg-brand-200 dark:bg-brand-700 dark:text-brand-300 dark:hover:bg-brand-600"
                >
                  {t('game.modals.addGame.select')}
                </button>
              </div>
              <p className="mt-1 text-xs text-brand-500">{t('game.modals.addGame.coverImageHint')}</p>
            </div>

            <div>
              <label className="mb-2 block text-sm font-medium text-brand-900 dark:text-white">{t('game.modals.addGame.developer')}</label>
              <input
                type="text"
                value={manualCompany}
                onChange={e => setManualCompany(e.target.value)}
                className="box-border block w-full rounded-lg border border-brand-300 bg-brand-50 p-3 text-brand-900 dark:border-brand-600 dark:bg-brand-700 dark:text-white"
              />
            </div>

            <div>
              <label className="mb-2 block text-sm font-medium text-brand-900 dark:text-white">{t('game.modals.addGame.summary')}</label>
              <textarea
                value={manualSummary}
                onChange={e => setManualSummary(e.target.value)}
                rows={3}
                className="box-border block w-full rounded-lg border border-brand-300 bg-brand-50 p-3 text-brand-900 dark:border-brand-600 dark:bg-brand-700 dark:text-white resize-none"
              />
            </div>

            <div className="flex justify-end space-x-4 pt-2">
              <button
                onClick={() => setStep(1)}
                className="rounded-lg border border-brand-300 px-5 py-2.5 text-sm font-medium text-brand-700 hover:bg-brand-100 dark:border-brand-600 dark:text-brand-300 dark:hover:bg-brand-700"
              >
                {t('game.modals.addGame.back')}
              </button>
              <button
                onClick={handleManualSave}
                disabled={!gameName || isLoading}
                className="rounded-lg bg-neutral-600 px-5 py-2.5 text-sm font-medium text-white hover:bg-neutral-700 disabled:opacity-50"
              >
                {isLoading ? t('game.modals.addGame.saving') : t('game.modals.addGame.save')}
              </button>
            </div>
          </div>
        )}
      </div>
    </div>
  );
}
