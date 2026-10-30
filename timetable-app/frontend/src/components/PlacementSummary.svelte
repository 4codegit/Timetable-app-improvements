<script>
        export let report;
        export let lessons;
</script>

<div class="panel" class:ok={report.unplaced.length === 0}>
        <div class="panel-head">
                <h3>Панель расстановки</h3>
                <span class="panel-pct" class:bad={report.unplaced.length > 0}>{report.placedPct}%</span>
        </div>
        <div class="bar"><div class="bar-fill" class:bad={report.unplaced.length > 0} style="width:{report.placedPct}%"></div></div>
        <div class="panel-stats">
                <span>Размещено <b>{report.placedHours}</b> из <b>{report.totalHours}</b> ч</span>
                {#if report.missingHours > 0}<span class="warn">осталось {report.missingHours} ч</span>{:else}<span class="good">все уроки размещены</span>{/if}
                <span class="muted">уроков в плане: {lessons.length}</span>
        </div>
        {#if report.reasonGroups.length}
                <div class="panel-reasons">
                        <b>Почему не расставлено ({report.unplaced.length} уроков):</b>
                        {#each report.reasonGroups as group}
                                <div class="reason-row">
                                        <span class="reason-badge">{group.items.length}</span>
                                        <span class="reason-text">{group.reason}</span>
                                        <span class="reason-who">{group.items.map((item) => item.subject + " · " + item.cls).join(", ")}</span>
                                </div>
                        {/each}
                </div>
        {/if}
        {#if report.classStats.length}
                <details class="panel-classes">
                        <summary>По классам: размещено / нужно недельных часов</summary>
                        <div class="cls-grid">
                                {#each report.classStats as classStat}
                                        <div class="cls-cell" class:bad={classStat.missing > 0} title={classStat.missing > 0 ? `не хватает ${classStat.missing}ч` : "полностью размещён"}>
                                                <span class="cls-name">{classStat.name}</span>
                                                <span class="cls-val">{classStat.placed}/{classStat.need}</span>
                                        </div>
                                {/each}
                        </div>
                </details>
        {/if}
</div>
