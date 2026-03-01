
import { models } from "../../../wailsjs/go/models";
import { GetImageBackupByUrl } from "../../../wailsjs/go/service/ImageService";
import { useEffect, useState } from "react";


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
                setImageUrl(res.local_path);
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
    const getLocalPath = () => {
        console.log("local path 01:", imageBackup.local_path)
        if (!imageBackup.local_path) {
            return ""
        }
        const ar = imageBackup.local_path.split("\\")
        if (ar.length <2) {
            return ""
        }
        const path = `/local/${ar[ar.length - 2]}/${ar[ar.length - 1]}`
        return path;
    }
    return (
        <div className="image-card">
            { imageBackup.local_path !== "" ? 
            (<img src={getLocalPath()} alt="Image" />) : 
            ( <img src={imageBackup.url} alt="Image" /> )}
            
        </div>
    );
}