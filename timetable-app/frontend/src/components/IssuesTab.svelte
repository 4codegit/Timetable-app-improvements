<script>
        export let vm;
</script>

<section class="card">
        <div class="card-head"><h2>🐛 Проблемы расписания</h2></div>
        <div class="issues-summary">
                <div class="issue-stat"><span class="issue-num">{vm.report.unplaced.length}</span><span class="issue-label">уроков не размещено</span></div>
                <div class="issue-stat"><span class="issue-num">{vm.report.conflicts.length}</span><span class="issue-label">накладок (учитель/класс/кабинет)</span></div>
                <div class="issue-stat"><span class="issue-num">{vm.report.violations.length}</span><span class="issue-label">нарушений жёстких ограничений</span></div>
                <div class="issue-stat"><span class="issue-num">{vm.report.placedPct}%</span><span class="issue-label">заполнено сетки</span></div>
        </div>

        {#if vm.report.unplaced.length > 0}
                <div class="issues-section">
                        <h3>❌ Не размещённые уроки ({vm.report.unplaced.length})</h3>
                        <p class="hint">Эти уроки из учебного плана не попали ни в одну ячейку расписания. Чаще всего причина — нет свободных слотов у учителя или класса, или жёсткие ограничения блокируют все варианты.</p>
                        <table class="data">
                                <thead><tr><th>Предмет</th><th>Класс</th><th>Учитель</th><th>Часов в неделю</th><th>Причина</th></tr></thead>
                                <tbody>
                                        {#each vm.report.unplaced as item}
                                                <tr class="issue-row">
                                                        <td>{vm.subjName(vm.subjects, item.subject_id)}</td>
                                                        <td>{vm.className(vm.classes, item.class_id)}</td>
                                                        <td>{vm.teachName(vm.teachers, item.teacher_id)}</td>
                                                        <td>{item.hours_per_week}</td>
                                                        <td class="issue-reason">{item.reason}</td>
                                                </tr>
                                        {/each}
                                </tbody>
                        </table>
                </div>
        {:else}
                <div class="issues-section empty-state">
                        <h3>✅ Все уроки размещены</h3>
                        <p class="hint">В расписании нет нераспределённых уроков.</p>
                </div>
        {/if}

        {#if vm.report.conflicts.length > 0}
                <div class="issues-section">
                        <h3>⚠️ Накладки ({vm.report.conflicts.length})</h3>
                        <p class="hint">Одна и та же сущность (учитель, класс или кабинет) стоит в двух местах одновременно.</p>
                        <table class="data">
                                <thead><tr><th>День</th><th>Пара</th><th>Тип</th><th>Участники</th></tr></thead>
                                <tbody>
                                        {#each vm.report.conflicts as conflict}
                                                <tr>
                                                        <td>{vm.dayName(conflict.day)}</td>
                                                        <td>П{conflict.slot + 1}</td>
                                                        <td>{conflict.type}</td>
                                                        <td>{#each conflict.items as item, i}{item.subject} ({item.who}){#if i < conflict.items.length - 1}, {/if}{/each}</td>
                                                </tr>
                                        {/each}
                                </tbody>
                        </table>
                </div>
        {/if}

        {#if vm.report.violations.length > 0}
                <div class="issues-section">
                        <h3>🚫 Нарушения жёстких ограничений ({vm.report.violations.length})</h3>
                        <p class="hint">Урок стоит в ячейке, которая запрещена ограничением (учитель занят, класс занят, кабинет занят).</p>
                        <table class="data">
                                <thead><tr><th>День</th><th>Пара</th><th>Что нарушено</th></tr></thead>
                                <tbody>
                                        {#each vm.report.violations as violation}
                                                <tr>
                                                        <td>{vm.dayName(violation.day)}</td>
                                                        <td>П{violation.slot + 1}</td>
                                                        <td>{violation.what}</td>
                                                </tr>
                                        {/each}
                                </tbody>
                        </table>
                </div>
        {/if}

        {#if vm.report.classStats.length > 0}
                <div class="issues-section">
                        <h3>📊 По классам</h3>
                        <table class="data">
                                <thead><tr><th>Класс</th><th>Размещено ч</th><th>Нужно ч</th><th>Не хватает</th><th>Статус</th></tr></thead>
                                <tbody>
                                        {#each vm.report.classStats as classStat}
                                                <tr class:bad={classStat.missing > 0}>
                                                        <td>{classStat.name}</td>
                                                        <td>{classStat.placed}</td>
                                                        <td>{classStat.need}</td>
                                                        <td class:bad={classStat.missing > 0}>{classStat.missing > 0 ? `-${classStat.missing}` : "—"}</td>
                                                        <td>{classStat.missing > 0 ? "не полностью" : "полностью"}</td>
                                                </tr>
                                        {/each}
                                </tbody>
                        </table>
                </div>
        {/if}

        {#if vm.report.teacherStats.length > 0}
                <div class="issues-section">
                        <h3>👨‍🏫 По учителям (нагрузка)</h3>
                        <p class="hint">Сколько часов из учебного плана размещено у каждого учителя. Нет лимита — только фактическая нагрузка.</p>
                        <table class="data">
                                <thead><tr><th>Учитель</th><th>Размещено ч</th></tr></thead>
                                <tbody>
                                        {#each vm.report.teacherStats as teacher}
                                                <tr>
                                                        <td>{teacher.name}</td>
                                                        <td>{teacher.placed} ч</td>
                                                </tr>
                                        {/each}
                                </tbody>
                        </table>
                </div>
        {/if}

        <div class="issues-actions" role="group" aria-label="Действия с расписанием">
                <button class="primary" on:click={vm.generate}>🔄 Перегенерировать расписание</button>
                <button on:click={vm.onGoSchedule}>↩ К расписанию</button>
        </div>
</section>
