import { Card, CardActionArea, CardContent, Typography } from "@mui/material";
import { styleNodeSettingsCard } from "./style";

interface INodeSettingsCard {

    title: string;
    handleClick: () => void
}

function NodeSettingsCard({ title, handleClick }: INodeSettingsCard) {
    return (
        <Card sx={styleNodeSettingsCard}>
            <CardActionArea onClick={handleClick} sx={{ height: '100%' }}>
                <CardContent>
                    <Typography gutterBottom variant="h5" component="div" margin={0}>
                        {title}
                    </Typography>
                </CardContent>
            </CardActionArea>
        </Card>
    )
}

export default NodeSettingsCard;