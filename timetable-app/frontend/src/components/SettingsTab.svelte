<script>
        export let days = 0;
        export let slots = 0;
        export let onSlotsChange;
        export let saveSettings;
        export let downloadRefsJSON;
        export let importRefsJSON;
        export let bellPeriods = [];
        export let normalizeTime;
</script>

<section class="card">
        <div class="card-head"><h2>Настройки школы</h2></div>
        <div class="row">
                <label>Учебных дней: <input type="number" min="1" max="7" bind:value={days} /></label>
                <label>Уроков в день: <input type="number" min="1" max="14" bind:value={slots} on:change={onSlotsChange} /></label>
                <button class="primary" on:click={saveSettings}>Сохранить</button>
                <span class="csvbar">
                        <button class="mini" on:click={() => downloadRefsJSON('periods')}>⬇ JSON звонков</button>
                        <label class="mini file">⬆<input type="file" accept=".json" on:change={(e) => importRefsJSON('periods', e)} /></label>
                </span>
        </div>
        <h3>Расписание звонков</h3>
        <table class="data">
                <thead><tr><th>Урок</th><th>Начало</th><th>Конец</th></tr></thead>
                <tbody>
                        {#each Array(slots) as _, si}
                                <tr>
                                        <td>П{si + 1}</td>
                                        <td><input class="bell-time" type="text" inputmode="numeric" placeholder="08:00" bind:value={bellPeriods[si].start} on:change={() => { bellPeriods[si].start = normalizeTime(bellPeriods[si].start); }} title="24-часовой формат ЧЧ:ММ" /></td>
                                        <td><input class="bell-time" type="text" inputmode="numeric" placeholder="08:45" bind:value={bellPeriods[si].end} on:change={() => { bellPeriods[si].end = normalizeTime(bellPeriods[si].end); }} title="24-часовой формат ЧЧ:ММ" /></td>
                                </tr>
                        {/each}
                </tbody>
        </table>
        <p class="hint">Время отображается в шапке расписания и в PDF. Хранится в базе (SQLite).</p>
</section>
