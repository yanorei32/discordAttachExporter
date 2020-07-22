# discordAttachExporter
Download attach files from [Tyrrrz/DiscordChatExporter](https://github.com/Tyrrrz/DiscordChatExporter) JSON

![image](https://user-images.githubusercontent.com/11992915/88126994-aef2a480-cc0d-11ea-81c3-89be311cf1d3.png)

# How to

## 1. Write `configure.yml` in executable dir

### Maximum parallel HTTP request count (required, greater than 1)
```yaml
parallelDownload: 6
```

### Limits by file-extension (optional)

The default configure is limit for images.

```yaml
downloadExtension:
  - png
  - jpg
  - jpeg
  - bmp
  - webp
  - gif
  - tiff
  - psd
  - ai
  - svg
```

## 2. Execute

```
discordAttachExporter <JSON by DiscordChatExporter>
```

And it downloads attachment files to the current directory.

Format:
```
discord_[Message TimeStamp in LocalTime]_[UserID]_[OriginalFileName]
```

Example:
```
discord_2020-06-07_20-06-06.357_000000000000000000_DSC_0020.JPG
```
