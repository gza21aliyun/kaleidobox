import { models } from "../../../wailsjs/go/models";
import { GetImageBackupByUrl } from "../../../wailsjs/go/service/ImageService";
import { useEffect, useState } from "react";
import { createPortal } from "react-dom";


interface ImageCardProps {
  url: string;
  className?: string;
  alt?: string;
}
export function ImageCard({
    url,
    className,
    alt
}: ImageCardProps) {
    const [imageUrl, setImageUrl] = useState("");

    useEffect(() => {
        GetImageBackupByUrl(url).then((res) => {
            if (res.local_path !== "") {
                setImageUrl(getLocalPath(res.local_path));
            } else if (res.url !== "") {
                setImageUrl(res.url);
            }
        });
        return () => {
            setImageUrl("");
         };
    }, [url]);



    return (
        <div className="image-card">
            { imageUrl === "" ? (<p>Loading...</p>) : 
            ( <img src={imageUrl} alt="Image"
                className={className}
             /> )}
            
        </div>
    );
}

interface ImageBackupProps {
  imageBackup: models.ImageBackup;
}

export function ImageBackupCard({
    imageBackup
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
                (<img src={getImageUrl()} alt="Image" />) : 
                ( <img src={imageBackup.url} alt="Image" /> )}
                
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
        console.log("local path 01:", localPath)
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