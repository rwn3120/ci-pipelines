(function (global) {

    "use strict";

    const SOURCE_URI = "/";

    function fetchWrapper(uri) {
        return new Promise((resolve, reject) => {
            fetch(uri)
                .then(response => {
                    if (response.status < 200 || response.status >= 300) {
                        reject(response);
                    } else {
                        resolve(response.json());
                    }
                });
        });
    }

    function loadData() {
        return fetchWrapper(SOURCE_URI);
    }

    function diffTime(now, previous) {

        const msPerMinute = 60 * 1000;
        const msPerHour = msPerMinute * 60;
        const msPerDay = msPerHour * 24;
        const msPerMonth = msPerDay * 30;
        const msPerYear = msPerDay * 365;

        const elapsed = now - previous.getTime();

        if (elapsed < msPerMinute) {
            return `${Math.round(elapsed / 1000)} seconds ago`;
        } else if (elapsed < msPerHour) {
            return `${Math.round(elapsed / msPerMinute)} minutes ago`;
        } else if (elapsed < msPerDay) {
            return `${Math.round(elapsed / msPerHour)} hours ago`;
        } else if (elapsed < msPerMonth) {
            return `${Math.round(elapsed / msPerDay)} days ago`;
        } else if (elapsed < msPerYear) {
            return `${Math.round(elapsed / msPerMonth)} months ago`;
        }
        return `${Math.round(elapsed / msPerYear)} years ago`;

    }

    function durationToText(durationInSeconds, includeSeconds = false) {

        durationInSeconds = parseInt(durationInSeconds, 10); // don't forget the second param
        const hours = Math.floor(durationInSeconds / 3600);
        const minutes = Math.floor((durationInSeconds - (hours * 3600)) / 60);
        const seconds = durationInSeconds - (hours * 3600) - (minutes * 60);

        const arr = [];

        if (hours > 0) {
            if (hours === 1) {
                arr.push("1 hour");
            } else {
                arr.push(`${hours} hours`);
            }
        }

        if (minutes > 0) {
            if (minutes === 1) {
                arr.push("1 minute");
            } else {
                arr.push(`${minutes} minutes`);
            }
        }

        if (seconds > 0 && includeSeconds) {
            if (seconds === 1) {
                arr.push("1 second");
            } else {
                arr.push(`${seconds} seconds`);
            }
        }
        return arr.join(" ");

    }

    function Tile({
        branchName,
        pipelineStatus,
        pipelineAuthor,
        pipelineDuration,
        pipelineStartedOn,
        pipelineUrl,
        pipelineId,
        pipelineAvatar
    } = {}) {
        this.branchName = branchName;
        this.pipelineStatus = pipelineStatus;
        this.pipelineAuthor = pipelineAuthor;
        this.pipelineDuration = pipelineDuration;
        this.pipelineStartedOn = pipelineStartedOn;
        this.pipelineId = pipelineId;
        this.pipelineUrl = pipelineUrl;
        this.pipelineAvatar = pipelineAvatar;

    }

    Tile.prototype.toHtml = function (now = Date.now()) {
        return `<div class='tile flex-item flex-md-3 flex-lg-2 flex-sm-12'><div class='tile-wrapper status-${this.pipelineStatus.toLowerCase()}' ><h5 class='ellipsis branch' title='${this.branchName}'>${this.branchName}</h5><h3 class='status ellipsis'><a target='_blank' href='${this.pipelineUrl}'>${this.pipelineStatus}</a></h3>
        <div>
            <span class='duration'>${durationToText(this.pipelineDuration) || "&nbsp;"}</span>
        </div>
        <div class='bottom-wrapper'>
        <div title='${this.pipelineStartedOn.toLocaleString()}' class='left startedon'>${diffTime(now, this.pipelineStartedOn)}</div>
        <div class='right avatar' title='${this.pipelineAuthor}' style='background-image: url(${this.pipelineAvatar})'></div>
        <div class='clear'></div>
        </div>
        </div> 
        </div>`;
    };

    function isProjectEmpty(project) {
        if (!project.branches || project.branches.length < 1) {
            return true;
        }
        return !project.branches.some(branch => branch.pipelines && branch.pipelines.length > 0);
    }

    function renderProject(project) {
        const heading = `<h2>Project: <a target='_blank' href='${project.web_url}'>${project.name}</a></h2>`;
        if (project.error || isProjectEmpty(project)) {
            return "";
        }

        const now = Date.now();
        const tiles = project.branches.reduce((progress, branch) => {
            if (!branch.pipelines || branch.pipelines.length < 1) {
                return progress;
            }
            const [latestPipeline] = branch.pipelines;
            return progress + new Tile({
                branchName: branch.branch,
                pipelineStatus: latestPipeline.status,
                pipelineAuthor: latestPipeline.User.name,
                pipelineAvatar: latestPipeline.User.avatar_url,
                pipelineDuration: latestPipeline.duration,
                pipelineStartedOn: new Date(latestPipeline.started_at),
                pipelineId: latestPipeline.id,
                pipelineUrl: latestPipeline.web_url || `${project.web_url}/pipelines/${latestPipeline.id}`
            }).toHtml(now);
        }, "");

        return `${heading}<div class='project flex'>${tiles}</div>`;
    }

    let timeout = 1000;

    function render(jsonData) {
        if (jsonData.length < 1) {
            timeout = 1000;
            return;
        }
        timeout = 30000;
        document.querySelector("#content").innerHTML = jsonData.reduce((progress, project) => progress + renderProject(project), "");
    }

    function start() {
        loadData()
            .then(render)
            .catch(err => global.console.log(err))
            .then(() => {
                setTimeout(start, timeout);
            });
    }

    start();

})(window);