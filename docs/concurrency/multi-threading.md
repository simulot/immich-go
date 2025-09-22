# Abstract 

The test  consists of uploading 2,846 items to an Immich server using the `immich-go` tool with varying numbers of concurrent processes. The goal is to analyze how the number of concurrent uploads affects the overall upload time and system resource usage.

The test system's CPU has 6 physical cores but 12 logical CPUs, thanks to Hyper-Threading technology.

As the number of concurrent processes increases from 1 to 6, the task completion time decreases significantly as each process gets a dedicated physical core. The performance gain continues between 6 and 9 processes because the system starts using the second logical thread on each core, utilizing idle resources to improve efficiency.

Beyond 12 concurrent processes (the number of logical cores), the performance gains diminish, and the time can even slightly increase. This is likely due to the overhead of the operating system needing to manage and switch between more processes than there are logical cores.
            
            
# System Information

## CPU Details

| Property            | Value                                         |
|---------------------|-----------------------------------------------|
| Architecture        | x86_64                                        |
| CPU op-mode(s)      | 32-bit, 64-bit                                |
| Address sizes       | 39 bits physical, 48 bits virtual             |
| Byte Order          | Little Endian                                 |
| CPU(s)              | 12                                            |
| Vendor ID           | GenuineIntel                                  |
| Model name          | Intel(R) Core(TM) i7-10750H CPU @ 2.60GHz     |
| CPU family          | 6                                             |
| Model               | 165                                           |
| Thread(s) per core  | 2                                             |
| Core(s) per socket  | 6                                             |
| Socket(s)           | 1                                             |
| Stepping            | 2                                             |
| CPU max MHz         | 5000.00                                       |
| CPU min MHz         | 800.00                                        |
| BogoMIPS            | 5199.98                                       |

---

# Performance Tests

The table below summarizes the upload performance for 2,846 items using varying numbers of concurrent processes:

| Concurrent Processes | Real Time   | User Time   | Sys Time    |
|---------------------:|:-----------|:------------|:------------|
| 1 (Baseline)         | 2m15.894s  | 0m28.587s   | 0m21.053s   |
| 2                    | 1m23.660s  | 0m28.489s   | 0m21.773s   |
| 5                    | 1m13.339s  | 0m33.579s   | 0m29.538s   |
| 6                    | 0m59.056s  | 0m32.258s   | 0m25.384s   |
| 9                    | 0m53.401s  | 0m31.836s   | 0m23.953s   |
| 12                   | 0m50.978s  | 0m31.978s   | 0m24.865s   |
| 18                   | 0m52.683s  | 0m33.811s   | 0m25.185s   |
| 24                   | 0m50.230s  | 0m33.574s   | 0m24.492s   |
| 48                   | 0m52.204s  | 0m32.865s   | 0m25.004s   |

**Notes:**
- Each test uploaded 2,846 items.
- *Real Time* is the total elapsed wall-clock time.
- *User Time* and *Sys Time* represent CPU time spent in user and system space, respectively.

---

