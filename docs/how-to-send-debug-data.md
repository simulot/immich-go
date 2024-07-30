# How to share data with the developer?

The structure of the takeout archive can be weird enough to get `Immich-go` confused.


In most of the cases, the list of files is sufficient for trouble-shooting the problem. 
This size of the list is much smaller than the full archive and contains enough information for simulating the import process.

This list reveals only the files's name and size, and the albums' name. 

If you agree, you can share it with me via a DM on discord @simulot.



## Get the file list from a zip takeout

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
   604784  2024-05-23 19:31   Takeout/Google Photos/Photos from 2018/IMG_8131.JPG
   645224  2024-05-23 19:31   Takeout/Google Photos/Photos from 2018/IMG_8130.JPG
   188804  2024-05-23 19:31   Takeout/Google Photos/Photos from 2018/IMG_8132.JPG
   375981  2024-05-23 19:31   Takeout/Google Photos/Photos from 2018/IMG_8129.JPG
   478073  2024-05-23 19:31   Takeout/Google Photos/Photos from 2018/IMG_8128.JPG
  2047609  2024-05-23 19:31   Takeout/Google Photos/Photos from 2018/IMG_8125.JPG
  2250833  2024-05-23 19:31   Takeout/Google Photos/Photos from 2018/IMG_8124.JPG
   429040  2024-05-23 19:31   Takeout/Google Photos/Photos from 2018/IMG_8120.JPG
   908856  2024-05-23 19:31   Takeout/Google Photos/Photos from 2018/IMG_8117.JPG
   699546  2024-05-23 19:31   Takeout/Google Photos/Photos from 2018/IMG_8118.JPG
   625635  2024-05-23 19:31   Takeout/Google Photos/Photos from 2018/IMG_8115.JPG
  1006873  2024-05-23 19:31   Takeout/Google Photos/Photos from 2018/IMG_8116.JPG
   499507  2024-05-23 19:31   Takeout/Google Photos/Photos from 2018/IMG_8114.JPG
 43189565  2024-05-23 19:31   Takeout/Google Photos/Photos from 2018/VID_20180819_192245.mp4
   541875  2024-05-23 19:32   Takeout/Google Photos/Photos from 2018/IMG_8112.JPG
   503405  2024-05-23 19:32   Takeout/Google Photos/Photos from 2018/IMG_8113.JPG
  1070437  2024-05-23 19:32   Takeout/Google Photos/Photos from 2018/IMG_8111.JPG
   583809  2024-05-23 19:32   Takeout/Google Photos/Photos from 2018/IMG_8110.JPG
   808994  2024-05-23 19:32   Takeout/Google Photos/Photos from 2018/IMG_20180718_163816.jpg
   798787  2024-05-23 19:32   Takeout/Google Photos/Photos from 2018/IMG_20180718_163817.jpg
...
```



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
-rw-r--r-- 0/0             838 2023-12-09 16:30 Takeout/Google Photos/Photos from 2023/PXL_20231207_202513366.jpg.json
-rw-r--r-- 0/0             827 2023-12-09 16:30 Takeout/Google Photos/Photos from 2023/PXL_20231209_074450784.LS.mp4.json
-rw-r--r-- 0/0         1453060 2023-12-09 16:30 Takeout/Google Photos/Photos from 2023/PXL_20231207_202525504.jpg
-rw-r--r-- 0/0             819 2023-12-09 16:30 Takeout/Google Photos/Photos from 2023/PXL_20231207_192200378.PORTRAIT.jpg.json
-rw-r--r-- 0/0             849 2023-12-09 16:30 Takeout/Google Photos/Photos from 2023/PXL_20231207_192157945.PORTRAIT.jpg.json
-rw-r--r-- 0/0         2852580 2023-12-09 16:30 Takeout/Google Photos/Photos from 2023/PXL_20231207_192125032.PORTRAIT.jpg
-rw-r--r-- 0/0             827 2023-12-09 16:30 Takeout/Google Photos/Photos from 2023/PXL_20231209_073951854.LS.mp4.json
-rw-r--r-- 0/0         3046592 2023-12-09 16:30 Takeout/Google Photos/Photos from 2023/PXL_20231207_192127213.PORTRAIT.jpg
-rw-r--r-- 0/0          684979 2023-12-09 16:30 Takeout/Google Photos/Photos from 2023/PXL_20231209_074450784.LS.mp4
-rw-r--r-- 0/0         2638469 2023-12-09 16:30 Takeout/Google Photos/Photos from 2023/PXL_20231207_192128472.PORTRAIT.jpg
-rw-r--r-- 0/0             819 2023-12-09 16:30 Takeout/Google Photos/Photos from 2023/PXL_20231207_192201288.PORTRAIT.jpg.json
-rw-r--r-- 0/0         1046367 2023-12-09 16:30 Takeout/Google Photos/Photos from 2023/PXL_20231207_202513366.jpg
-rw-r--r-- 0/0             867 2023-12-09 16:30 Takeout/Google Photos/Photos from 2023/PXL_20231207_192128472.PORTRAIT.jpg.json
-rw-r--r-- 0/0          602708 2023-12-09 16:30 Takeout/Google Photos/Photos from 2023/PXL_20231207_191742033.jpg
-rw-r--r-- 0/0             867 2023-12-09 16:30 Takeout/Google Photos/Photos from 2023/PXL_20231207_192127213.PORTRAIT.jpg.json
-rw-r--r-- 0/0             867 2023-12-09 16:30 Takeout/Google Photos/Photos from 2023/PXL_20231207_192146687.PORTRAIT.jpg.json
-rw-r--r-- 0/0             867 2023-12-09 16:30 Takeout/Google Photos/Photos from 2023/PXL_20231207_192125032.PORTRAIT.jpg.json
...
```