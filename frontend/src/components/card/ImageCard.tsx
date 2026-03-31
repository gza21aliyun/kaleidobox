import { models } from "../../../wailsjs/go/models";
import { GetImageBackupByUrl } from "../../../wailsjs/go/service/ImageService";
import { useEffect, useState, useRef } from "react";
import { createPortal } from "react-dom";




interface ImageBackupProps {
  imageBackup: models.ImageBackup;
  className?: string | undefined;
  alt?: string;
  style?: React.CSSProperties | undefined;
  draggable?: boolean | undefined;
  referrerPolicy?: React.HTMLAttributeReferrerPolicy | undefined;  
    onDragStart?: React.DragEventHandler | undefined;
    onError?: React.ReactEventHandler | undefined;
}

export function ImageBackupCard({
    imageBackup, className, alt, style, draggable, referrerPolicy, onDragStart, onError
}: ImageBackupProps) { 
    const [isModalOpen, setIsModalOpen] = useState(false);
    const [scale, setScale] = useState(1);

    const getImageUrl = () => {
        if (imageBackup.local_path !== "") {
            return getLocalPath(imageBackup.local_path);
        }
        return imageBackup.url;
    };

    const handleWheel = (e: React.WheelEvent<HTMLDivElement>) => {
        e.preventDefault();
        const delta = e.deltaY > 0 ? 0.9 : 1.1;
        setScale(prev => Math.max(0.1, Math.min(5, prev * delta)));
    };

    const handleKeyDown = (e: KeyboardEvent) => {
        if (e.key === 'ArrowUp' || e.key === 'ArrowRight') {
            setScale(prev => Math.min(5, prev * 1.1));
            e.preventDefault();
        } else if (e.key === 'ArrowDown' || e.key === 'ArrowLeft') {
            setScale(prev => Math.max(0.1, prev * 0.9));
            e.preventDefault();
        } else if (e.key === 'Escape') {
            setIsModalOpen(false);
            e.preventDefault();
        }
    };

    useEffect(() => {
        if (isModalOpen) {
            document.addEventListener('keydown', handleKeyDown);
            return () => {
                document.removeEventListener('keydown', handleKeyDown);
                setScale(1);
            };
        }
    }, [isModalOpen]);

    return (
        <>
            <div className="image-card cursor-pointer hover:opacity-80 transition-opacity" onClick={() => setIsModalOpen(true)}>
                { imageBackup.local_path !== "" ? 
                (<img src={getImageUrl()} 
                alt={alt || "Image"}
                className={className}
                style={style}
                draggable={false}
                onDragStart={onDragStart}
                referrerPolicy={referrerPolicy}
                onError={onError}
                 />) : 
                ( <img src={imageBackup.url} alt={alt || "Image"}
                    className={className}
                    style={style}         
                    draggable={false}
                    onDragStart={onDragStart}
                    referrerPolicy={referrerPolicy}   
                    onError={onError}    
                /> )}
                
            </div>

            {isModalOpen && createPortal(
                <div 
                    className="fixed inset-0 bg-black/80 z-50 flex items-center justify-center p-4"
                    onClick={() => setIsModalOpen(false)}
                    onWheel={handleWheel}
                >
                    <div 
                        className="relative transition-all duration-200"
                        style={{ 
                            transform: `scale(${scale})`, 
                            transformOrigin: 'center center',
                            maxWidth: '90vw',
                            maxHeight: '90vh'
                        }}
                        onClick={(e) => e.stopPropagation()}
                    >
                        <button
                            className="absolute -top-10 right-0 text-white hover:text-gray-300 text-3xl font-bold"
                            onClick={() => setIsModalOpen(false)}
                        >
                            ×
                        </button>
                        <img 
                            src={getImageUrl()} 
                            alt="Image" 
                            className="max-w-full max-h-[90vh] object-contain"
                        />
                    </div>
                </div>,
                document.body
            )}
        </>
    );
}

function getLocalPath(localPath: string)  {
        // console.log("local path 01:", localPath)
        if (!localPath) {
            return ""
        }
        const ar = localPath.split("\\")
        if (ar.length < 3) {
            return ""
        }
        const path = `/local/${ar[ar.length - 3]}/${ar[ar.length - 2]}/${ar[ar.length - 1]}`
        return path;
    }


interface ImageCardProps {
  url: string;
  className?: string;
  alt?: string;
  style?: React.CSSProperties;
  draggable?: boolean | undefined;
  referrerPolicy?: React.HTMLAttributeReferrerPolicy | undefined;  
    onDragStart?: React.DragEventHandler | undefined;
    onError?: React.ReactEventHandler | undefined;
  lazyLoad?: boolean;
}
export function ImageCard({
    url,
    className,
    alt,
    style,
    draggable,
    referrerPolicy,
    onDragStart,
    onError,
    lazyLoad = true
}: ImageCardProps) {
    const [imageBackup, setImageBackup] = useState<models.ImageBackup | null>(null);
    const [loading, setLoading] = useState(false);
    const [isVisible, setIsVisible] = useState(!lazyLoad); // 非懒加载时默认可见
    const ref = useRef<HTMLDivElement>(null);
    const [mounted, setMounted] = useState(false);

    // 组件挂载完成
    useEffect(() => {
        setMounted(true);
        return () => setMounted(false);
    }, []);

    // 监听元素是否进入可视区域（仅在懒加载时启用）
    useEffect(() => {
        if (!lazyLoad) return;
        
        const observer = new IntersectionObserver(
            ([entry]) => {
                if (entry.isIntersecting) {
                    setIsVisible(true);
                }
            },
            { 
                threshold: 0.01, // 减小阈值，只要有1%进入视口就触发
                rootMargin: '50px' // 添加预加载区域，提前50px开始加载
            }
        );

        if (ref.current) {
            observer.observe(ref.current);
            
            // 初始状态检查：立即检查元素是否在可视区域内
            const checkVisibility = () => {
                if (!ref.current) return;
                
                const rect = ref.current.getBoundingClientRect();
                const isInView = (
                    rect.top < (window.innerHeight || document.documentElement.clientHeight) &&
                    rect.bottom > 0 &&
                    rect.left < (window.innerWidth || document.documentElement.clientWidth) &&
                    rect.right > 0
                );
                
                if (isInView) {
                    setIsVisible(true);
                }
            };
            
            // 立即检查
            checkVisibility();
            
            // 延迟再次检查，确保DOM已经完全渲染
            const timer = setTimeout(checkVisibility, 100);
            
            return () => {
                clearTimeout(timer);
                if (ref.current) {
                    observer.unobserve(ref.current);
                }
            };
        }

        return () => {
            if (ref.current) {
                observer.unobserve(ref.current);
            }
        };
    }, [lazyLoad]);

    // 加载图片逻辑
    useEffect(() => {
        if (lazyLoad && !mounted) return;
        if ((lazyLoad && !isVisible) || !url || url === "") {
            return;
        }
        setLoading(true);
        
        const fetchImage = async () => {
            
            try {
                const res = await GetImageBackupByUrl(url, true);
                if (mounted) {
                    setImageBackup(res);
                    setLoading(false);
                }
            } catch (error) {
                if (mounted) {
                    setLoading(false);
                }
            }
        };
        
        fetchImage();

        return () => {
            setImageBackup(null);
            setLoading(false);
        };
    }, [url, isVisible, mounted]);

    // 渲染逻辑
    const renderContent = () => {
        if (lazyLoad && !isVisible) {
            // 元素未进入可视区域时，显示占位符
            return (
                <div ref={ref} className="bg-gray-100 rounded-md" style={{ ...style, minHeight: style?.height ? `${parseInt(style.height.toString()) * 3}px` : '120px' }}>
                </div>
            );
        }

        if (loading) {
            // 加载中，显示loading动画
            return (
                <div ref={ref} className="image-card-loading flex items-center justify-center bg-gray-100 rounded-md" style={{ ...style, minHeight: style?.height ? `${parseInt(style.height.toString()) * 3}px` : '120px' }}>
                    <div className="w-8 h-8 border-4 border-gray-200 border-t-blue-500 rounded-full animate-spin"></div>
                </div>
            );
        }

        if (!imageBackup) {
            // 加载失败或无数据，显示占位符
            return (
                <div ref={ref} className="bg-gray-100 rounded-md" style={{ ...style, minHeight: style?.height ? `${parseInt(style.height.toString()) * 3}px` : '120px' }}>
                </div>
            );
        }

        // 加载成功，显示图片
        return (
            <div ref={ref}>
                <ImageBackupCard 
                    imageBackup={imageBackup} 
                    alt={alt}
                    className={className}
                    style={style}
                    draggable={draggable}
                    onDragStart={onDragStart}
                    referrerPolicy={referrerPolicy}
                    onError={onError}
                    />
            </div>
        );
    };

    return renderContent();
}


//     const [imageBackup, setImageBackup] = useState<models.ImageBackup | null>(null);
//     const [loading, setLoading] = useState(true);

//     useEffect(() => {
//         setLoading(true);
//         GetImageBackupByUrl(url, true).then((res) => {
//             setImageBackup(res);
//             setLoading(false);
//         }).catch(() => {
//             setLoading(false);
//         });
//         return () => {
//             setImageBackup(null);
//             setLoading(false);
//          };
//     }, [url]);

//     if (loading) {
//         return (
//             <div className="image-card-loading flex items-center justify-center bg-gray-100 rounded-md" style={{ ...style, minHeight: style?.height ? `${parseInt(style.height.toString()) * 3}px` : '120px' }}>
//                 <div className="w-8 h-8 border-4 border-gray-200 border-t-blue-500 rounded-full animate-spin"></div>
//             </div>
//         );
//     }

//     if (!imageBackup) {
//         return null;
//     }

//     return (
//         <ImageBackupCard 
//             imageBackup={imageBackup} 
//             alt={alt}
//             className={className}
//             style={style}
//             draggable={draggable}
//             onDragStart={onDragStart}
//             referrerPolicy={referrerPolicy}
//             onError={onError}
//             />
//     );
// }