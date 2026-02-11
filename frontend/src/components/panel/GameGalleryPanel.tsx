import { models } from "../../../wailsjs/go/models";

interface GameGalleryPanelProps {
  game: models.Game;
}

export function GameGalleryPanel({ game }: GameGalleryPanelProps) {
  // 检查是否有图片且过滤条件满足
  const hasImages = game.images && game.images.length > 0;
  
  if (!hasImages) {
    return null;
  }

  // 处理图片数组，过滤封面图和特定后缀的图片
  const galleryImages = game.images
    .split(",")
    .filter(img => img !== game.cover_url && !img.endsWith("pl.jpg"))
    .filter(img => img.trim() !== ""); // 过滤空字符串

  // 如果没有符合条件的图片，不显示面板
  if (galleryImages.length === 0) {
    return null;
  }

  return (
    <div className="flex flex-col gap-2">
      <div className="font-semibold mb-2 text-brand-900 dark:text-white">画廊</div>
      <div className="grid grid-cols-3 gap-2">
        {galleryImages.map((image, index) => (
          <img
            key={index}
            src={image}
            alt={`Gallery image ${index + 1}`}
            className="w-full h-auto object-cover rounded cursor-pointer hover:opacity-80 transition-opacity"
            onError={(e) => {
              // 图片加载失败时的处理
              const target = e.target as HTMLImageElement;
              target.style.display = 'none';
            }}
          />
        ))}
      </div>
    </div>
  );
}