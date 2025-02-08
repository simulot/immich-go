# How to share data with the developer?

The structure of the takeout archive can be weird enough to get `Immich-go` confused.


In most of the cases, the list of files is sufficient for trouble-shooting the problem. 
This size of the list is much smaller than the full archive and contains enough information for simulating the import process.

This list reveals only the files's name and size, and the albums' name. 

If you agree, you can share it with me via a DM on discord @simulot.


## Get the file list from a zip takeout under linux / macos / wsl

```sh
for f in *.zip; do echo "Part: $f"; unzip -l $f; done >list.lst
```

This produces a file like this:
```
Part: takeout-20240523T170453Z-001.zip: 
Archive:  takeout-20240523T170453Z-001.zip
  Length      Date    Time    Name
---------  ---------- -----   ----
   800432  2024-05-23 19:31   Takeout/Google Photos/Photos from 2018/IMG_8135.JPG
  1166223  2024-05-23 19:31   Takeout/Google Photos/Photos from 2018/IMG_8133.JPG
 17132148  2024-05-23 19:31   Takeout/Google Photos/Photos from 2018/VID_20180819_191954.mp4
...
```

## Get the file list from a zip takeout under windows


### **Step 1: Download the Script**
1. **Visit the GitHub Repository**:
   - [Go to the GitHub page](/docs) where the `ZipContents.ps1` script is hosted.
2. **Download the Script**:
   - Locate the `ZipContents.ps1` file on the page.
   - Click the file name to view its contents.
   - Click the **Raw** button to display the raw script.
   - Right-click anywhere on the page and select **Save As**.
   - Save the file as `ZipContents.ps1` to a known folder (e.g., `Downloads`).

### **Step 2: Open a terminal**
1. **Open  the file explorer**:
   - Navigate to the location where the script is downloaded
   - Right-click on the folder and select either **Open PowerShell window here** or **Open in terminal**.


### **Step 4: Run the Script**
Run the script with the folder containing ZIP files and the desired output file path as parameters:
```powershell
powershell -ExecutionPolicy Bypass -File .\ZipContents.ps1 -Path "C:\Path\To\Zips" -OutputFile "C:\Path\To\Zips\ZipContents.txt"
```

- Replace `C:\Path\To\Zips` with the folder containing the ZIP files you want to process.

### **Step 5: Send the ZipContents.txt file to the project team**
1. Once the script completes, locate the output file (e.g., `ZipContents.txt`).
2. Send the file to the project team @simulot for further analysis.

### **Step 6: Clean Up**
1. Delete the script (`ZipContents.ps1`) if you no longer need it.


## Get the file list from a tgz takeout

```sh
for f in *.tgz; do echo "Part: $f"; tar -tzvf $f; done >list.lst
```

This produces a file like this:
```
Part: takeout-20231209T153001Z-001.tgz: 
-rw-r--r-- 0/0         3987330 2023-12-09 16:30 Takeout/Google Photos/Photos from 2023/PXL_20231207_192201288.PORTRAIT.jpg
-rw-r--r-- 0/0         3825143 2023-12-09 16:30 Takeout/Google Photos/Photos from 2023/PXL_20231207_192200378.PORTRAIT.jpg
-rw-r--r-- 0/0             838 2023-12-09 16:30 Takeout/Google Photos/Photos from 2023/PXL_20231207_202525504.jpg.json
-rw-r--r-- 0/0         4136113 2023-12-09 16:30 Takeout/Google Photos/Photos from 2023/PXL_20231207_192157945.PORTRAIT.jpg
-rw-r--r-- 0/0         2817334 2023-12-09 16:30 Takeout/Google Photos/Photos from 2023/PXL_20231207_192146687.PORTRAIT.jpg
...
```


