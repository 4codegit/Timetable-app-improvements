<script>
        export let DAY_NAMES = [];
        export let activeDayIdx = [];
        export let schoolDaysMask = 0;
        export let toggleSchoolDay;
        export let consPickKind = "teacher";
        export let setPickKind;
        export let consPickId = 0;
        export let consPickList = [];
        export let consPickEntity = null;
        export let pickName;
        export let slots = 0;
        export let periodLabel;
        export let dayName;
        export let isForbidden;
        export let toggleForbidden;
        export let constraints = [];
        export let constraintSummary;
        export let removeConstraint;
        export let subjects = [];
        export let ruleSubject = 0;
        export let ruleMax = 2;
        export let addRunRule;
</script>

<section class="card">
        <div class="card-head"><h2>Ограничения</h2></div>
        <div class="school-days">
                <span class="field-label">Учебные дни недели:</span>
                {#each DAY_NAMES as dn, d}
                        <label class="chk" title={activeDayIdx.includes(d) ? "Учебный день — снять галочку, чтобы убрать день из расписания" : "Выходной — включить день в расписание"}>
                                <input type="checkbox"
                                        checked={(schoolDaysMask & (1 << d)) !== 0}
                                        on:change={(e) => toggleSchoolDay(d, e.currentTarget.checked)} /> {dn}
                        </label>
                {/each}
        </div>
        <p class="hint">Снятые дни сразу исчезают из сетки расписания, генерации и PDF — как выходные. Последний оставшийся день снять нельзя.</p>

        <div class="pick-bar">
                <div class="pick-kind">
                        <button class:active={consPickKind === "teacher"} on:click={() => setPickKind("teacher")}>Учитель</button>
                        <button class:active={consPickKind === "class"} on:click={() => setPickKind("class")}>Класс</button>
                        <button class:active={consPickKind === "room"} on:click={() => setPickKind("room")}>Кабинет</button>
                </div>
                <select value={consPickEntity ? consPickEntity.id : 0} on:change={(e) => consPickId = +e.currentTarget.value}>
                        {#each consPickList as x (x.id)}<option value={x.id}>{pickName(x)}</option>{/each}
                </select>
                <span class="hint pick-hint">Клик по ячейке — запретить или разрешить это время.</span>
        </div>
        <table class="pick-grid">
                <thead><tr><th class="d"></th>{#each Array(slots) as _, si}<th>{#if periodLabel(si)}{periodLabel(si)}{:else}П{si + 1}{/if}</th>{/each}</tr></thead>
                <tbody>
                        {#each activeDayIdx as d (d)}
                                <tr>
                                        <td class="day">{dayName(d)}</td>
                                        {#each Array(slots) as _, si}
                                                <td class="slot" class:forbidden={isForbidden(d, si)}
                                                        on:click={() => toggleForbidden(d, si)}
                                                        title={isForbidden(d, si) ? "Клик — разрешить" : "Клик — запретить"}>{#if isForbidden(d, si)}✕{/if}</td>
                                        {/each}
                                </tr>
                        {/each}
                </tbody>
        </table>

        <ul class="list constraints-list">
                {#each constraints as x}
                        <li>
                                <span class="con-summary">{constraintSummary(x)}</span>
                                <button class="danger sm" on:click={() => removeConstraint(x.id)}>✕</button>
                        </li>
                {/each}
        </ul>

        <div class="rule-form">
                <span class="field-label">Правило:</span>
                <select bind:value={ruleSubject}>
                        <option value={0}>Все предметы</option>
                        {#each subjects as s}<option value={s.id}>{s.name}</option>{/each}
                </select>
                <label class="rule-max">не более <input type="number" min="1" max="8" bind:value={ruleMax} /> подряд в день</label>
                <button class="primary sm" on:click={addRunRule}>+ Правило</button>
        </div>
        <p class="hint">Как в aSc Timetables: ученики не получают один и тот же предмет N уроков подряд (например, информатика 1-й, 2-й и 3-й уроки одной строкой). Половинки делёного урока в одном слоте считаются одним периодом.</p>
</section>
