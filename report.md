# Report for assignment 3

This is a template for your report. You are free to modify it as needed. It is not required to use markdown for your report either, but the report has to be delivered in a standard, cross-platform format.

## Project

Name: immich-go

URL: [https://github.com/simulot/immich-go](https://github.com/simulot/immich-go)

One or two sentences describing it: It is an open-source tool to upload large photo collections to a self-hosted Immich server. It is designed to streamline the uploading.

## Onboarding experience

Did it build and run as documented?

See the assignment for details; if everything works out of the box, there is no need to write much here. If the first project(s) you picked ended up being unsuitable, you can describe the "onboarding experience" for each project, along with reason(s) why you changed to a different one.   

*(a) Did you have to install a lot of additional tools to build the software?*   
We just had to download Go (Golang) which was simple and we then had everything we needed. All the other packages are handled by Golangs module system.

*(b) Were those tools well documented?*   
Yes, Go is very well documented. 

*(c) Were other components installed automatically by the build script?*   
Yes, they were downloaded automatically by the go mod system. However, these are only installed in the project repository and not globally on our system.

*(d) Did the build conclude automatically without errors?*   
Yes

*(e) How well do examples and tests run on your system(s)?*  
After setting up the server to upload the pictures to and running the needed commands, the test went without errors and uploaded a picture.   
We can run all the existing test with ´go test ./…´ and they run smoothly in a couple of seconds and all pass

*Do you plan to continue or choose another project?*  
We plan to continue with the project.  

## Complexity

1. *What are your results for five complex functions?*  
   * *Did all methods (tools vs. manual count) get the same result?*  
   * *Are the results clear?*

All results are listed below [Go to Results from CC calculations](#results-from-cc-calculations)

2. *Are the functions just complex, or also long?*

Some of the functions like [`ParseDir`](#parsedir)  is both very cyclomatically complex (CC \= 64\) and also quite long (NLOC \= 241). But this is not true for all our chosen functions, for example [`NewLocalFiles`](#newlocalfiles) only have 35 NLOC but still quite high CC at 12\.

3. *What is the purpose of the functions?*  
- [`NewLocalFiles`](#newlocalfiles): 

The NewLocalFiles function creates a LocalAssetBrowser object for handling local files. It also checks for conflicting settings, sets up configurations, date handling, album management and tags etcetera. It then creates needed collectors or helpers and returns a configured browser.  

- [`ParseDir`](#parsedir): 

The function systematically scans a directory (and optionally its subdirectories), identifies media files, filters unwanted content, extracts metadata, and organizes files into groups for further processing—likely for import into a media management system.

- [`WriteAsset`](#writeasset): 

The **[`WriteAsset`](#writeasset)** function is responsible for writing a media asset (like a photo or video) and its metadata files (XMP, JSON) to a specific location in a filesystem. It ensures that directories exist, handles filename conflicts, and writes additional sidecar metadata files if needed.

- [`OpenClient`](#openclient):

The **[`OpenClient`](#openclient)** function is responsible for initializing the client within the application. The function handles configuration of server and necessary files for logging and API tracing if enabled.  
	

- [`FilterAsset`](#filterasset):  

The **[`FilterAsset`](#filterasset)** function is responsible for filtering assets based on user-defined criteria. The flags (filter) are set by the user and used in getAssets from the same file, which will use filterAsset to compare the asset with the specific flags (filter criterias).  

4. *Are exceptions taken into account in the given measurements?*

There were no exceptions in the code to take into account since golang does not have traditional exceptions. Error handling in Go is based upon returning errors through functions and panic if we need to crash the program. So in our measurements for the CC, we count each error return as an exit point

5. Is the documentation clear w.r.t. all the possible outcomes?

There is no documentation about possible outcomes for each functions. This is probably due to the golang community preferring concise readable code above excessive commenting. The documentation that exists are in the README of the project and they are aimed at the main command of the CLI (upload, from-folder, archive etc.). This documentation is meant to explain how to use the program (CLI), not specifically to explain possible outcomes of the used functions.

## Results from CC calculations

**Difference between our CC and lizard**  
We think that lizard doesn't count the number of return points and just have that value at a static 1\. This would result in lizards CCN being always number of decision points in the code \+ 2 \- 1 which checks out in all of our calculations. We used the formula M \= E \- N \+ 2, where M is the CC, E is decision points and N is return points. 

First function: In readFolder.go function [`NewLocalFiles`](#newlocalfiles)   
NLOC \= 35  
Results from Lizard: CCN \= 13  
Our results \= 12  
The difference is how we calculate the return. We calculate with \- 2 returns but we think Lizard counts with only one return. We used the formula from the slides that said “Number of exit points”. 

Second function: In readFolder.go function [`ParseDir`](#parsedir)   
NLOC \= 241  
Results from Lizard: CCN \= 31 \+ 39 \+ 2 \= 72  
Our results \= 64

- decision points: 71  
- returns: 9

The difference is how we calculate the return. We calculate with \- 2 returns but we think Lizard counts with only one return. We used the formula from the slides that said “Number of exit points”. 

Third function: In client.go in app folder, function [`OpenClient`](#openclient)  
NLOC \= 51  
Results from Lizard: CCN \= 14  
Our results \= 7

- decision points: 13  
- returns: 8

This one had a big difference from Lizard, we still think it is something with the `return` because this function had 8 returns. 

Fourth function: In writefolder.go in adapter/folder, function [`WriteAsset`](#writeasset)  
NLOC \= 81  
Results from Lizard: CCN \= 17  
Our results \= 9

- decision points: 16  
- returns: 9

Fifth function: In fromimmich.go in adapter/fromimmich, function [`FilterAsset`](#filterasset)  
NLOC \= 67  
Results from Lizard: CCN \= 22  
Our results \= 14

- decision points: 21  
- returns: 9

## Refactoring

*Estimated impact of refactoring (lower CC, but other drawbacks?).*

***Plan for refactoring complex code:***  
[`NewLocalFiles`](#newlocalfiles) :   
The function has a lot of if cases to handle what should happen with the LocalAssetBrowser depending on which flags we specify in the function arguments. One simple way of refactoring this function is to collect all of these if cases regarding setting flags and move them to a separate function, for example called “LaFlagConfigurator”. This would severely lower the CC of [`NewLocalFiles`](#newlocalfiles) at the cost of turning one function with high CC to 2 functions with medium CC.

[`ParseDir`](#parsedir):   
It could be possible to extract the go-function starting on line 273 inside ParseDir to lower the CC of ParseDir. A possible drawback is then the need to send the data forth and back, it could be possible to send it as pointers to avoid needing to duplicate data, thus reducing memory usage.  

[`OpenClient`](#openclient):   
One of the tasks of this function is to configure the logging. This is done primarily with the code from row (54-72). This functionality is possible to move to a separate function, for example “ConfigureLogging”, which would reduce the CC of OpenClient by a few points. The drawback is once again that we need to maintain two functions and some data has to be transferred back and forth.

[`WriteAsset`](#writeasset):   
On rows 93-121 in the function, a for-loop is used to handle if the file's name already exists and then rename it with an added index. To reduce the CC in the WriteAsset function, we could write a separate function for it and call it if needed. The drawback is the need for another function and once again, the need to maintain two functions instead of one, and handle data transferred. 

[`FilterAsset`](#filterasset):   
There is a part of the function that handles filtering of the albums. Since the albums are not used in another part of the function it would make sense to refactor this into a separate function, “filterAlbums” or “handleAlbums” or something similar. This could probably be run in parallel as well. This will reduce the CC of the filterAsset function, but a drawback would be the need for another function that clutters the file a bit. It could be confusing since all filters are currently collected in the filterAsset function. Perhaps in the future it will become more complicated and in need of more refactoring and then this refactor would make more sense. 

## Coverage

### Tools

*Document your experience in using a "new"/different coverage tool.*

*How well was the tool documented? Was it possible/easy/difficult to integrate it with your build environment?*  
We used the cover tool that is already built into Go. It was well-documented and easy to build into our environment.   
Steps:

* Generate coverprofile from tests  
  * Run \`go test \-coverprofile=coverage.out ./…\`  
* Generate .txt file from coverprofile to see coverage for each function  
  * Run \`go tool cover \-func=coverage.out \> coverageOut.txt\`  
* Generate browseabale coverprofile  
  * Run \`go tool cover \-html=coverage.out \-o coverage.html\`

### Your own coverage tool

Show a patch (or link to a branch) that shows the instrumented code to gather coverage measurements.  
Our coverage tool is in the branch `coverage`, [https://github.com/adam-viberg/immich-go/tree/coverage](https://github.com/adam-viberg/immich-go/tree/coverage). 

The patch is probably too long to be copied here, so please add the git command that is used to obtain the patch instead:  
`git checkout coverage`

### Evaluation

1. How detailed is your coverage measurement?

Our coverage measurements for our chosen functions are relatively detailed. We chose not to count && or || as separate branches since that would require rewriting of the source code so that would perhaps cause it to not be 100% detailed. Furthermore we chose to have the output of our coverage tool to be in a text file which maybe is not the best way to do it, but it is entirely possible to create a data structure containing branch-coverage information.

2. What are the limitations of your own tool?

As previously mentioned one of the limitations could be that the output is in a text file rather than a data structure like an array. However this could also be a positive since it is very easy to browse the results of our coverage measurements. Another limitation is that our tool does not give a coverage percentage of each measured function, it instead gives the names (IDs) of the covered branches. If you would want a percentage you would have to do the math yourself which is a small limitation.

3. Are the results of your tool consistent with existing coverage tools?

The results are very consistent with the cover tool we used from Go. In the browsable html profile we can see which branches are covered by the tests and these align with the branches we print out in our coverageBranches.txt


## Coverage improvement

Show the comments that describe the requirements for the coverage.

Report of old coverage: \[link\]  
In the branch `coverage` do the following commands:   
Steps:

* Generate coverprofile from tests  
  * Run \`go test \-coverprofile=coverage.out ./…\`  
* Generate .txt file from coverprofile to see coverage for each function  
  * Run \`go tool cover \-func=coverage.out \> coverageOut.txt\`  
* Generate browseabale coverprofile  
  * Run \`go tool cover \-html=coverage.out \-o coverage.html\`

Report of new coverage: \[link\]

Test cases added: Two test cases per the five functions we have discussed above. 

git diff ...

Number of test cases added: two per team member (P)

## Self-assessment: Way of working

Current state according to the Essence standard:   
We feel that we are still in the state of “In Place”. We feel that we are on the way to the state of “Working Well” but are just not there fully yet. We discussed it as a group and felt that even though we are fulfilling some of the points of “Working Well”, we are not fulfilling them all. We also feel that it is a bit too soon to be in that state, considering how long we have been working as a group. One of the doubts we have of the “Working Well” state is the item “The team naturally applies the practices without thinking about them.”, we are still reminding each other to use issues and other practices. Although we feel that we have improved a lot since we started like knowing each other's strengths and dividing the work based on that. We work a lot more fluently as a group now and we think the group works well together. We still have improvements to make to be a fully “Working Well” group like improving our communication as we had some communication problems regarding when and where to have meetings this week. 

## Overall experience

*What are your main take-aways from this project? What did you learn?*  
One big takeaway from this project was finding and evaluating an open-source project based on the requirements we had for the project. For most of us, we have also worked in Go for the first time during this project. We have learnt about coverage and how to manually calculate cyclomatic complexity for a function. As well as how to write tests for a function in Go.  

Is there something special you want to mention here?  
It was fun working with the group but we prefer writing our projects instead of reading others code. 

## Contributions

Adam Viberg: [`ParseDir`](#parseDir)  
Max Andreasen: [`FilterAsset`](#filterasset)  
Måns Zellman: [`NewLocalFiles`](#newlocalfiles)  
Olivia Håkans: [`WriteAsset`](#writeasset)  
Samvel Hovhannisyan: [`OpenClient`](#openclient)


## Appendix (Functions)

### NewLocalFiles
```go
func NewLocalFiles(ctx context.Context, l *fileevent.Recorder, flags *ImportFolderOptions, fsyss ...fs.FS) (*LocalAssetBrowser, error) {


	coverageTester.WriteUniqueLine("NewLocalFiles - Branch 0/7 Covered")

	if flags.ImportIntoAlbum != "" && flags.UsePathAsAlbumName != FolderModeNone {

		coverageTester.WriteUniqueLine("NewLocalFiles - Branch 1/7 Covered")
		return nil, errors.New("cannot use both --into-album and --folder-as-album")
	}

	la := LocalAssetBrowser{
		fsyss: fsyss,
		flags: flags,
		log:   l,
		pool:  worker.NewPool(10), // TODO: Make this configurable
		requiresDateInformation: flags.InclusionFlags.DateRange.IsSet() ||
			flags.TakeDateFromFilename || flags.StackBurstPhotos ||
			flags.ManageHEICJPG != filters.HeicJpgNothing || flags.ManageRawJPG != filters.RawJPGNothing,
	}

	if flags.PicasaAlbum {
		coverageTester.WriteUniqueLine("NewLocalFiles - Branch 2/7 Covered")
		la.picasaAlbums = gen.NewSyncMap[string, PicasaAlbum]() // make(map[string]PicasaAlbum)
	}

	if flags.InfoCollector == nil {
		coverageTester.WriteUniqueLine("NewLocalFiles - Branch 3/7 Covered")
		flags.InfoCollector = filenames.NewInfoCollector(flags.TZ, flags.SupportedMedia)
	}

	if flags.InclusionFlags.DateRange.IsSet() {
		coverageTester.WriteUniqueLine("NewLocalFiles - Branch 4/7 Covered")
		flags.InclusionFlags.DateRange.SetTZ(flags.TZ)
	}

	if flags.SessionTag {
		coverageTester.WriteUniqueLine("NewLocalFiles - Branch 5/7 Covered")
		flags.session = fmt.Sprintf("{immich-go}/%s", time.Now().Format("2006-01-02 15:04:05"))
	}

	// if flags.ExifToolFlags.UseExifTool {
	// 	err := exif.NewExifTool(&flags.ExifToolFlags)
	// 	if err != nil {
	// 		return nil, err
	// 	}
	// }

	if flags.ManageEpsonFastFoto {
		coverageTester.WriteUniqueLine("NewLocalFiles - Branch 6/7 Covered")
		g := epsonfastfoto.Group{}
		la.groupers = append(la.groupers, g.Group)
	}
	if flags.ManageBurst != filters.BurstNothing {
		coverageTester.WriteUniqueLine("NewLocalFiles - Branch 7/7 Covered")
		la.groupers = append(la.groupers, burst.Group)
	}
	la.groupers = append(la.groupers, series.Group)

	return &la, nil
}
```

### ParseDir
```go
func (la *LocalAssetBrowser) parseDir(ctx context.Context, fsys fs.FS, dir string, gOut chan *assets.Group) error {
	coverageTester.WriteUniqueLine("parseDir - Branch 0/53 Covered")

	fsName := ""
	if fsys, ok := fsys.(interface{ Name() string }); ok {
		coverageTester.WriteUniqueLine("parseDir - Branch 1/53 Covered")
		fsName = fsys.Name()
	}

	var as []*assets.Asset
	var entries []fs.DirEntry
	var err error

	select {
	case <-ctx.Done():
		coverageTester.WriteUniqueLine("parseDir - Branch 2/53 Covered")
		return ctx.Err()
	default:
		entries, err = fs.ReadDir(fsys, dir)
		if err != nil {
			coverageTester.WriteUniqueLine("parseDir - Branch 3/53 Covered")
			return err
		}
	}

	for _, entry := range entries {
		base := entry.Name()
		name := path.Join(dir, base)
		if entry.IsDir() {
			coverageTester.WriteUniqueLine("parseDir - Branch 4/53 Covered")
			continue
		}

		if la.flags.BannedFiles.Match(name) {
			coverageTester.WriteUniqueLine("parseDir - Branch 5/53 Covered")
			la.log.Record(ctx, fileevent.DiscoveredDiscarded, fshelper.FSName(fsys, entry.Name()), "reason", "banned file")
			continue
		}

		if la.flags.SupportedMedia.IsUseLess(name) {
			coverageTester.WriteUniqueLine("parseDir - Branch 6/53 Covered")
			la.log.Record(ctx, fileevent.DiscoveredUseless, fshelper.FSName(fsys, entry.Name()))
			continue
		}

		if la.flags.PicasaAlbum && (strings.ToLower(base) == ".picasa.ini" || strings.ToLower(base) == "picasa.ini") {
			coverageTester.WriteUniqueLine("parseDir - Branch 7/53 Covered")
			a, err := ReadPicasaIni(fsys, name)
			if err != nil {
				coverageTester.WriteUniqueLine("parseDir - Branch 8/53 Covered")
				la.log.Record(ctx, fileevent.Error, fshelper.FSName(fsys, name), "error", err.Error())
			} else {
				coverageTester.WriteUniqueLine("parseDir - Branch 9/53 Covered")
				la.picasaAlbums.Store(dir, a) // la.picasaAlbums[dir] = a
				la.log.Log().Info("Picasa album detected", "file", fshelper.FSName(fsys, path.Join(dir, name)), "album", a.Name)
			}
			continue
		}

		ext := filepath.Ext(base)
		mediaType := la.flags.SupportedMedia.TypeFromExt(ext)

		if mediaType == filetypes.TypeUnknown {
			coverageTester.WriteUniqueLine("parseDir - Branch 10/53 Covered")
			la.log.Record(ctx, fileevent.DiscoveredUnsupported, fshelper.FSName(fsys, name), "reason", "unsupported file type")
			continue
		}

		switch mediaType {
		case filetypes.TypeUseless:
			coverageTester.WriteUniqueLine("parseDir - Branch 11/53 Covered")
			la.log.Record(ctx, fileevent.DiscoveredUseless, fshelper.FSName(fsys, name))
			continue
		case filetypes.TypeImage:
			coverageTester.WriteUniqueLine("parseDir - Branch 12/53 Covered")
			la.log.Record(ctx, fileevent.DiscoveredImage, fshelper.FSName(fsys, name))
		case filetypes.TypeVideo:
			coverageTester.WriteUniqueLine("parseDir - Branch 13/53 Covered")
			la.log.Record(ctx, fileevent.DiscoveredVideo, fshelper.FSName(fsys, name))
		case filetypes.TypeSidecar:
			coverageTester.WriteUniqueLine("parseDir - Branch 14/53 Covered")
			if la.flags.IgnoreSideCarFiles {
				coverageTester.WriteUniqueLine("parseDir - Branch 15/53 Covered")
				la.log.Record(ctx, fileevent.DiscoveredDiscarded, fshelper.FSName(fsys, name), "reason", "sidecar file ignored")
				continue
			}
			la.log.Record(ctx, fileevent.DiscoveredSidecar, fshelper.FSName(fsys, name))
			continue
		}

		if !la.flags.InclusionFlags.IncludedExtensions.Include(ext) {
			coverageTester.WriteUniqueLine("parseDir - Branch 16/53 Covered")
			la.log.Record(ctx, fileevent.DiscoveredDiscarded, fshelper.FSName(fsys, name), "reason", "extension not included")
			continue
		}

		if la.flags.InclusionFlags.ExcludedExtensions.Exclude(ext) {
			coverageTester.WriteUniqueLine("parseDir - Branch 17/53 Covered")
			la.log.Record(ctx, fileevent.DiscoveredDiscarded, fshelper.FSName(fsys, name), "reason", "extension excluded")
			continue
		}

		select {
		case <-ctx.Done():
			coverageTester.WriteUniqueLine("parseDir - Branch 18/53 Covered")
			return ctx.Err()
		default:
			// we have a file to process
			a, err := la.assetFromFile(ctx, fsys, name)
			if err != nil {
				coverageTester.WriteUniqueLine("parseDir - Branch 19/53 Covered")
				la.log.Record(ctx, fileevent.Error, fshelper.FSName(fsys, name), "error", err.Error())
				return err
			}
			if a != nil {
				coverageTester.WriteUniqueLine("parseDir - Branch 20/53 Covered")
				as = append(as, a)
			}
		}
	}

	// process the left over dirs
	for _, entry := range entries {
		base := entry.Name()
		name := path.Join(dir, base)
		if entry.IsDir() {
			coverageTester.WriteUniqueLine("parseDir - Branch 21/53 Covered")
			if la.flags.BannedFiles.Match(name) {
				coverageTester.WriteUniqueLine("parseDir - Branch 22/53 Covered")
				la.log.Record(ctx, fileevent.DiscoveredDiscarded, fshelper.FSName(fsys, name), "reason", "banned folder")
				continue // Skip this folder, no error
			}
			if la.flags.Recursive && entry.Name() != "." {
				coverageTester.WriteUniqueLine("parseDir - Branch 23/53 Covered")
				la.concurrentParseDir(ctx, fsys, name, gOut)
			}
			continue
		}
	}

	in := make(chan *assets.Asset)
	go func() {
		defer close(in)

		sort.Slice(as, func(i, j int) bool {
			// Sort by radical first
			radicalI := as[i].Radical
			radicalJ := as[j].Radical
			if radicalI != radicalJ {
				coverageTester.WriteUniqueLine("parseDir - Branch 24/53 Covered")
				return radicalI < radicalJ
			}
			// If radicals are the same, sort by date
			return as[i].CaptureDate.Before(as[j].CaptureDate)
		})

		for _, a := range as {
			// check the presence of a JSON file
			jsonName, err := checkExistSideCar(fsys, a.File.Name(), ".json")
			if err == nil && jsonName != "" {
				coverageTester.WriteUniqueLine("parseDir - Branch 25/53 Covered")
				buf, err := fs.ReadFile(fsys, jsonName)
				if err != nil {
					coverageTester.WriteUniqueLine("parseDir - Branch 26/53 Covered")
					la.log.Record(ctx, fileevent.Error, nil, "error", err.Error())
				} else {
					if bytes.Contains(buf, []byte("immich-go version")) {
						coverageTester.WriteUniqueLine("parseDir - Branch 27/53 Covered")
						md := &assets.Metadata{}
						err = jsonsidecar.Read(bytes.NewReader(buf), md)
						if err != nil {
							coverageTester.WriteUniqueLine("parseDir - Branch 28/53 Covered")
							la.log.Record(ctx, fileevent.Error, nil, "error", err.Error())
						} else {
							md.File = fshelper.FSName(fsys, jsonName)
							a.FromApplication = a.UseMetadata(md) // Force the use of the metadata coming from immich export
							a.OriginalFileName = md.FileName      // Force the name of the file to be the one from the JSON file
						}
					} else {
						la.log.Log().Warn("JSON file detected but not from immich-go", "file", fshelper.FSName(fsys, jsonName))
					}
				}
			}
			// check the presence of a XMP file
			xmpName, err := checkExistSideCar(fsys, a.File.Name(), ".xmp")
			if err == nil && xmpName != "" {
				coverageTester.WriteUniqueLine("parseDir - Branch 29/53 Covered")
				buf, err := fs.ReadFile(fsys, xmpName)
				if err != nil {
					coverageTester.WriteUniqueLine("parseDir - Branch 30/53 Covered")
					la.log.Record(ctx, fileevent.Error, nil, "error", err.Error())
				} else {
					md := &assets.Metadata{}
					err = xmpsidecar.ReadXMP(bytes.NewReader(buf), md)
					if err != nil {
						coverageTester.WriteUniqueLine("parseDir - Branch 31/53 Covered")
						la.log.Record(ctx, fileevent.Error, nil, "error", err.Error())
					} else {
						md.File = fshelper.FSName(fsys, xmpName)
						a.FromSideCar = a.UseMetadata(md)
					}
				}
			}

			// Read metadata from the file only id needed (date range or take date from filename)
			if la.requiresDateInformation {
				coverageTester.WriteUniqueLine("parseDir - Branch 32/53 Covered")
				if a.CaptureDate.IsZero() {
					coverageTester.WriteUniqueLine("parseDir - Branch 33/53 Covered")
					// no date in XMP, JSON, try reading the metadata
					f, err := a.OpenFile()
					if err == nil {
						coverageTester.WriteUniqueLine("parseDir - Branch 34/53 Covered")
						md, err := exif.GetMetaData(f, a.Ext, la.flags.TZ)
						if err != nil {
							coverageTester.WriteUniqueLine("parseDir - Branch 35/53 Covered")
							la.log.Record(ctx, fileevent.INFO, a.File, "warning", err.Error())
						} else {
							a.FromSourceFile = a.UseMetadata(md)
						}
						if (md == nil || md.DateTaken.IsZero()) && !a.NameInfo.Taken.IsZero() && la.flags.TakeDateFromFilename {
							coverageTester.WriteUniqueLine("parseDir - Branch 36/53 Covered")
							// no exif, but we have a date in the filename and the TakeDateFromFilename is set
							a.FromApplication = &assets.Metadata{
								DateTaken: a.NameInfo.Taken,
							}
							a.CaptureDate = a.FromApplication.DateTaken
						}
						f.Close()
					}
				}
			}

			if !la.flags.InclusionFlags.DateRange.InRange(a.CaptureDate) {
				coverageTester.WriteUniqueLine("parseDir - Branch 37/53 Covered")
				a.Close()
				la.log.Record(ctx, fileevent.DiscoveredDiscarded, a.File, "reason", "asset outside date range")
				continue
			}

			// Add tags
			if len(la.flags.Tags) > 0 {
				coverageTester.WriteUniqueLine("parseDir - Branch 38/53 Covered")
				for _, t := range la.flags.Tags {
					a.AddTag(t)
				}
			}

			// Add folder as tags
			if la.flags.FolderAsTags {
				coverageTester.WriteUniqueLine("parseDir - Branch 39/53 Covered")
				t := fsName
				if dir != "." {
					coverageTester.WriteUniqueLine("parseDir - Branch 40/53 Covered")
					t = path.Join(t, dir)
				}
				if t != "" {
					coverageTester.WriteUniqueLine("parseDir - Branch 41/53 Covered")
					a.AddTag(t)
				}
			}

			// Manage albums
			if la.flags.ImportIntoAlbum != "" {
				coverageTester.WriteUniqueLine("parseDir - Branch 42/53 Covered")
				a.Albums = []assets.Album{{Title: la.flags.ImportIntoAlbum}}
			} else {
				done := false
				if la.flags.PicasaAlbum {
					coverageTester.WriteUniqueLine("parseDir - Branch 43/53 Covered")
					if album, ok := la.picasaAlbums.Load(dir); ok {
						coverageTester.WriteUniqueLine("parseDir - Branch 44/53 Covered")
						a.Albums = []assets.Album{{Title: album.Name, Description: album.Description}}
						done = true
					}
				}
				if !done && la.flags.UsePathAsAlbumName != FolderModeNone && la.flags.UsePathAsAlbumName != "" {
					coverageTester.WriteUniqueLine("parseDir - Branch 45/53 Covered")
					Album := ""
					switch la.flags.UsePathAsAlbumName {
					case FolderModeFolder:
						coverageTester.WriteUniqueLine("parseDir - Branch 46/53 Covered")
						if dir == "." {
							coverageTester.WriteUniqueLine("parseDir - Branch 47/53 Covered")
							Album = fsName
						} else {
							Album = filepath.Base(dir)
						}
					case FolderModePath:
						coverageTester.WriteUniqueLine("parseDir - Branch 48/53 Covered")
						parts := []string{}
						if fsName != "" {
							coverageTester.WriteUniqueLine("parseDir - Branch 49/53 Covered")
							parts = append(parts, fsName)
						}
						if dir != "." {
							coverageTester.WriteUniqueLine("parseDir - Branch 50/53 Covered")
							parts = append(parts, strings.Split(dir, "/")...)
							// parts = append(parts, strings.Split(dir, string(filepath.Separator))...)
						}
						Album = strings.Join(parts, la.flags.AlbumNamePathSeparator)
					}
					a.Albums = []assets.Album{{Title: Album}}
				}
			}

			if la.flags.SessionTag {
				coverageTester.WriteUniqueLine("parseDir - Branch 51/53 Covered")
				a.AddTag(la.flags.session)
			}
			select {
			case in <- a:
			case <-ctx.Done():
				coverageTester.WriteUniqueLine("parseDir - Branch 52/53 Covered")
				return
			}
		}
	}()

	gs := groups.NewGrouperPipeline(ctx, la.groupers...).PipeGrouper(ctx, in)
	for g := range gs {
		select {
		case gOut <- g:
		case <-ctx.Done():
			coverageTester.WriteUniqueLine("parseDir - Branch 53/53 Covered")
			return ctx.Err()
		}
	}
	return nil
}
```

### WriteAsset
```go
func (w *LocalAssetWriter) WriteAsset(ctx context.Context, a *assets.Asset) error {

	coverageTester.WriteUniqueLine("WriteAsset - Branch 0 (Main) Covered")

	base := a.Base
	dir := w.pathOfAsset(a)
	if _, ok := w.createdDir[dir]; !ok { // Branch 1
		coverageTester.WriteUniqueLine("WriteAsset - Branch 1 Covered of 16 possible")

		err := fshelper.MkdirAll(w.WriteToFS, dir, 0o755)
		if err != nil { // Branch 2
			coverageTester.WriteUniqueLine("WriteAsset - Branch 2 Covered of 16 possible")
			return err
		}
		w.createdDir[dir] = struct{}{}
	}
	select {
	case <-ctx.Done(): // Branch 3
		coverageTester.WriteUniqueLine("WriteAsset - Branch 3 Covered of 16 possible")
		return ctx.Err()
	default:
		r, err := a.OpenFile()
		if err != nil { // Branch 4
			coverageTester.WriteUniqueLine("WriteAsset - Branch 4 Covered of 16 possible")
			return err
		}
		defer r.Close()

		select {
		case <-ctx.Done(): // Branch 5
			coverageTester.WriteUniqueLine("WriteAsset - Branch 5 Covered of 16 possible")
			return ctx.Err()
		default:
			// Add an index to the file name if it already exists, or the XMP or JSON
			index := 0
			ext := path.Ext(base)
			radical := base[:len(base)-len(ext)]
			for { // Branch 6
				coverageTester.WriteUniqueLine("WriteAsset - Branch 6 Covered of 16  possible")
				if index > 0 { // Branch 7
					coverageTester.WriteUniqueLine("WriteAsset - Branch 7 Covered of 16 possible")
					base = fmt.Sprintf("%s~%d%s", radical, index, path.Ext(base))
				}
				_, err := fs.Stat(w.WriteToFS, path.Join(dir, base))
				if err == nil { // Branch 8
					coverageTester.WriteUniqueLine("WriteAsset - Branch 8 Covered of 16 possible")
					index++
					continue
				}
				_, err = fs.Stat(w.WriteToFS, path.Join(dir, base+".XMP"))
				if err == nil { // Branch 9
					coverageTester.WriteUniqueLine("WriteAsset - Branch 9 Covered of 16 possible")
					index++
					continue
				}
				_, err = fs.Stat(w.WriteToFS, path.Join(dir, base+".JSON"))
				if err == nil { // Branch 10
					coverageTester.WriteUniqueLine("WriteAsset - Branch 10 Covered of 16 possible")
					index++
					continue
				}
				break
			}

			// write the asset
			err = fshelper.WriteFile(w.WriteToFS, path.Join(dir, base), r)
			if err != nil { // Branch 11
				coverageTester.WriteUniqueLine("WriteAsset - Branch 11 Covered of 16 possible")
				return err
			}
			// XMP?
			if a.FromSideCar != nil { // Branch 12
				coverageTester.WriteUniqueLine("WriteAsset - Branch 12 Covered of 16 possible")
				// Sidecar file is set, copy it
				var scr fs.File
				scr, err = a.FromSideCar.File.Open()
				if err != nil { // Branch 13
					coverageTester.WriteUniqueLine("WriteAsset - Branch 13 Covered of 16 possible")
					return err
				}
				debugfiles.TrackOpenFile(scr, a.FromSideCar.File.Name())
				defer scr.Close()
				defer debugfiles.TrackCloseFile(scr)
				var scw fshelper.WFile
				scw, err = fshelper.OpenFile(w.WriteToFS, path.Join(dir, base+".XMP"), os.O_RDWR|os.O_CREATE|os.O_TRUNC, 0o644)
				if err != nil { // Branch 14
					coverageTester.WriteUniqueLine("WriteAsset - Branch 14 Covered of 16 possible")
					return err
				}
				_, err = io.Copy(scw, scr)
				scw.Close()
			}

			// Having metadata from an Application or immich-go JSON?
			if a.FromApplication != nil { // Branch 15
				coverageTester.WriteUniqueLine("WriteAsset - Branch 15 Covered of 16 possible")
				var scw fshelper.WFile
				scw, err = fshelper.OpenFile(w.WriteToFS, path.Join(dir, base+".JSON"), os.O_RDWR|os.O_CREATE|os.O_TRUNC, 0o644)
				if err != nil { // Branch 16
					coverageTester.WriteUniqueLine("WriteAsset - Branch 16 Covered of 16 possible")
					return err
				}
				err = jsonsidecar.Write(a.FromApplication, scw)
				scw.Close()
			}

			return err
		}
	}
}
```

### OpenClient
```go 
func OpenClient(ctx context.Context, cmd *cobra.Command, app *Application) error {
	coverageTester.WriteUniqueLine("OpenClient - Branch 0 covered")

	var err error
	client := app.Client()
	log := app.Log()

	if client.Server != "" {
		coverageTester.WriteUniqueLine("OpenClient - Branch 1/13 covered")
		client.Server = strings.TrimSuffix(client.Server, "/")
	}
	if client.TimeZone != "" {
		// Load the specified timezone
		coverageTester.WriteUniqueLine("OpenClient - Branch 2/13 covered")
		client.TZ, err = time.LoadLocation(client.TimeZone)
		if err != nil {
			coverageTester.WriteUniqueLine("OpenClient - Branch 3/13 covered")
			return err
		}
	}

	// Plug the journal on the Log
	if log.File != "" {
		coverageTester.WriteUniqueLine("OpenClient - Branch 4/13 covered")
		if log.mainWriter == nil {
			coverageTester.WriteUniqueLine("OpenClient - Branch 5/13 covered")
			err := configuration.MakeDirForFile(log.File)
			if err != nil {
				coverageTester.WriteUniqueLine("OpenClient - Branch 6/13 covered")
				return err
			}
			f, err := os.OpenFile(log.File, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o664)
			if err != nil {
				coverageTester.WriteUniqueLine("OpenClient - Branch 7/13 covered")
				return err
			}
			err = log.sLevel.UnmarshalText([]byte(strings.ToUpper(log.Level)))
			if err != nil {
				coverageTester.WriteUniqueLine("OpenClient - Branch 8/13 covered")
				return err
			}
			log.setHandlers(f, nil)
			// prepare the trace file name
			client.APITraceWriterName = strings.TrimSuffix(log.File, filepath.Ext(log.File)) + ".trace.log"
		}
	}

	err = client.Initialize(ctx, app)
	if err != nil {
		coverageTester.WriteUniqueLine("OpenClient - Branch 9/13 covered")
		return err
	}

	err = client.Open(ctx)
	if err != nil {
		coverageTester.WriteUniqueLine("OpenClient - Branch 10/13 covered")
		return err
	}

	if client.APITrace {
		coverageTester.WriteUniqueLine("OpenClient - Branch 11/13 covered")
		if client.APITraceWriter == nil {
			coverageTester.WriteUniqueLine("OpenClient - Branch 12/13 covered")
			client.APITraceWriter, err = os.OpenFile(client.APITraceWriterName, os.O_CREATE|os.O_WRONLY, 0o664)
			if err != nil {
				coverageTester.WriteUniqueLine("OpenClient - Branch 13/13 covered")
				return err
			}
			client.Immich.EnableAppTrace(client.APITraceWriter)
		}
		app.log.Message("Check the API-TRACE file: %s", client.APITraceWriterName)
	}
	return nil
}
```

### FilterAsset
```go
func (f *FromImmich) filterAsset(ctx context.Context, a *immich.Asset, grpChan chan *assets.Group) error {
	var err error

	// Checks if favourited asset and flag is favourite
	if f.flags.Favorite && !a.IsFavorite {
		coverageTester.WriteUniqueLine("Branch 1")
		return nil
	}

	// Probably filtering based on if photo is put in the bin
	if !f.flags.WithTrashed && a.IsTrashed {
		coverageTester.WriteUniqueLine("Branch 2")
		return nil
	}

	// Refactor the album section
	albums := immich.AlbumsFromAlbumSimplified(a.Albums)

	// Some filter set up in getAsset to determine is albums much be fetched later.
	if f.mustFetchAlbums && len(albums) == 0 {
		coverageTester.WriteUniqueLine("Branch 3")
		albums, err = f.flags.client.Immich.GetAssetAlbums(ctx, a.ID)
		if err != nil {
			coverageTester.WriteUniqueLine("Branch 4")
			return f.logError(err)
		}
	}

	// Checks if the asset is present in any albums by f.flags.Albums.
	if len(f.flags.Albums) > 0 && len(albums) > 0 {
		coverageTester.WriteUniqueLine("Branch 5")
		keepMe := false
		newAlbumList := []assets.Album{}
		for _, album := range f.flags.Albums {
			coverageTester.WriteUniqueLine("Branch 6")
			for _, aAlbum := range albums {
				coverageTester.WriteUniqueLine("Branch 7")
				if album == aAlbum.Title {
					coverageTester.WriteUniqueLine("Branch 8")
					keepMe = true
					newAlbumList = append(newAlbumList, aAlbum)
				}
			}
		}
		if !keepMe {
			coverageTester.WriteUniqueLine("Branch 9")
			return nil
		}
		albums = newAlbumList
	}

	// Some information are missing in the metadata result,
	// so we need to get the asset details
	a, err = f.flags.client.Immich.GetAssetInfo(ctx, a.ID)
	if err != nil {
		coverageTester.WriteUniqueLine("Branch 10")
		return f.logError(err)
	}
	asset := a.AsAsset()
	asset.SetNameInfo(f.ic.GetInfo(asset.OriginalFileName))
	asset.File = fshelper.FSName(f.ifs, a.ID)

	asset.FromApplication = &assets.Metadata{
		FileName:    a.OriginalFileName,
		Latitude:    a.ExifInfo.Latitude,
		Longitude:   a.ExifInfo.Longitude,
		Description: a.ExifInfo.Description,
		DateTaken:   a.ExifInfo.DateTimeOriginal.Time,
		Trashed:     a.IsTrashed,
		Archived:    a.IsArchived,
		Favorited:   a.IsFavorite,
		Rating:      byte(a.Rating),
		Albums:      albums,
		Tags:        asset.Tags,
	}

	// Filter on rating, unsure what kind of rating though.
	if f.flags.MinimalRating > 0 && a.Rating < f.flags.MinimalRating {
		coverageTester.WriteUniqueLine("Branch 11")
		return nil
	}

	// Filter asset on set date range.
	if f.flags.DateRange.IsSet() {
		coverageTester.WriteUniqueLine("Branch 12")
		if asset.CaptureDate.Before(f.flags.DateRange.After) || asset.CaptureDate.After(f.flags.DateRange.Before) {
			coverageTester.WriteUniqueLine("Branch 13")
			return nil
		}
	}

	// This part is used to channel the asset
	g := assets.NewGroup(assets.GroupByNone, asset)
	select {
	case grpChan <- g:
		coverageTester.WriteUniqueLine("Branch 14")
	case <-ctx.Done():
		coverageTester.WriteUniqueLine("Branch 15")
		return ctx.Err()
	}
	coverageTester.WriteUniqueLine("Branch 16")
	return nil
}
```
