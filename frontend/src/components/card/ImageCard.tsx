import { models } from "../../../wailsjs/go/models";
import { GetImageBackupByUrl } from "../../../wailsjs/go/service/ImageService";
import { useEffect, useState } from "react";
import { createPortal } from "react-dom";


interface ImageCardProps {
  url: string;
}
export function ImageCard({
    url
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
            ( <img src={imageUrl} alt="Image" /> )}
            
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

    const getImageUrl = () => {
        if (imageBackup.local_path !== "") {
            return getLocalPath(imageBackup.local_path);
        }
        return imageBackup.url;
    };

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
                >
                    <div 
                        className="relative max-w-7xl max-h-screen"
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